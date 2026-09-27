#include <windows.h>
#include <tlhelp32.h>

#include <cstdint>
#include <cstring>
#include <vector>

namespace {

constexpr size_t kPatchSize = 14;
constexpr size_t kMaxCapturedKeys = 64;
constexpr uintptr_t kWeixin41155HookRva = 0x342311F;
constexpr uintptr_t kWeixin41155FormatRva = 0x8BD45C4;
constexpr LONG kCaptureIdle = 0;
constexpr LONG kCaptureWriting = 1;
constexpr LONG kCaptureReady = 2;

BYTE* g_hook = nullptr;
BYTE g_original[kPatchSize] = {};
BYTE* g_stub = nullptr;

struct alignas(8) CaptureSlot {
    // State transitions are strictly idle -> writing -> ready -> idle.
    // The hook atomically claims idle before copying, then publishes ready.
    // The worker never reads the key while the slot is still being written.
    volatile LONG pending;
    DWORD reserved;
    BYTE key[32];
};

CaptureSlot* g_capture = nullptr;

void Emit(BYTE*& out, BYTE value) { *out++ = value; }

void EmitBytes(BYTE*& out, const BYTE* data, size_t size) {
    std::memcpy(out, data, size);
    out += size;
}

void EmitU64(BYTE*& out, uint64_t value) {
    std::memcpy(out, &value, sizeof(value));
    out += sizeof(value);
}

bool ModuleContainsRVA(HMODULE module, uintptr_t rva, size_t size) {
    const BYTE* base = reinterpret_cast<const BYTE*>(module);
    const auto* dos = reinterpret_cast<const IMAGE_DOS_HEADER*>(base);
    if (dos->e_magic != IMAGE_DOS_SIGNATURE || dos->e_lfanew <= 0 ||
        dos->e_lfanew > 0x1000) {
        return false;
    }
    const auto* nt = reinterpret_cast<const IMAGE_NT_HEADERS64*>(
        base + dos->e_lfanew);
    if (nt->Signature != IMAGE_NT_SIGNATURE ||
        nt->OptionalHeader.Magic != IMAGE_NT_OPTIONAL_HDR64_MAGIC) {
        return false;
    }
    const size_t imageSize = nt->OptionalHeader.SizeOfImage;
    return rva <= imageSize && size <= imageSize - rva;
}

std::vector<HANDLE> SuspendOtherThreads() {
    std::vector<HANDLE> suspended;
    const DWORD processID = GetCurrentProcessId();
    const DWORD currentThreadID = GetCurrentThreadId();
    HANDLE snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, 0);
    if (snapshot == INVALID_HANDLE_VALUE) {
        return suspended;
    }

    THREADENTRY32 entry = {};
    entry.dwSize = sizeof(entry);
    if (Thread32First(snapshot, &entry)) {
        do {
            if (entry.th32OwnerProcessID != processID ||
                entry.th32ThreadID == currentThreadID) {
                continue;
            }
            HANDLE thread = OpenThread(THREAD_SUSPEND_RESUME, FALSE,
                                       entry.th32ThreadID);
            if (thread == nullptr) {
                continue;
            }
            if (SuspendThread(thread) == static_cast<DWORD>(-1)) {
                CloseHandle(thread);
                continue;
            }
            suspended.push_back(thread);
        } while (Thread32Next(snapshot, &entry));
    }
    CloseHandle(snapshot);
    return suspended;
}

void ResumeThreads(std::vector<HANDLE>& threads) {
    for (auto it = threads.rbegin(); it != threads.rend(); ++it) {
        ResumeThread(*it);
        CloseHandle(*it);
    }
    threads.clear();
}

void SignalReadyEvent() {
    wchar_t name[256] = {};
    const DWORD length = GetEnvironmentVariableW(
        L"WECHAT_CLI_KEY_CAPTURE_READY_EVENT", name,
        static_cast<DWORD>(sizeof(name) / sizeof(name[0])));
    if (length == 0 || length >= sizeof(name) / sizeof(name[0])) {
        return;
    }
    HANDLE event = OpenEventW(EVENT_MODIFY_STATE, FALSE, name);
    if (event != nullptr) {
        SetEvent(event);
        CloseHandle(event);
    }
}

bool WriteCaptureFile(const BYTE* key) {
    wchar_t path[MAX_PATH * 4] = {};
    DWORD length = GetEnvironmentVariableW(
        L"WECHAT_CLI_KEY_CAPTURE_FILE", path,
        static_cast<DWORD>(sizeof(path) / sizeof(path[0])));
    if (length == 0 || length >= sizeof(path) / sizeof(path[0])) {
        return false;
    }

    static constexpr char hex[] = "0123456789abcdef";
    char encoded[64] = {};
    for (size_t i = 0; i < 32; ++i) {
        encoded[i * 2] = hex[key[i] >> 4];
        encoded[i * 2 + 1] = hex[key[i] & 0x0f];
    }

    HANDLE file = CreateFileW(path, FILE_APPEND_DATA, FILE_SHARE_READ, nullptr,
                              OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, nullptr);
    if (file == INVALID_HANDLE_VALUE) {
        return false;
    }
    DWORD written = 0;
    const BOOL keyOk = WriteFile(file, encoded, sizeof(encoded), &written, nullptr);
    static constexpr char newline = '\n';
    DWORD newlineWritten = 0;
    const BOOL newlineOk = WriteFile(file, &newline, 1, &newlineWritten, nullptr);
    FlushFileBuffers(file);
    CloseHandle(file);
    return keyOk && written == sizeof(encoded) && newlineOk && newlineWritten == 1;
}

bool AlreadyCaptured(const BYTE captured[][32], size_t count, const BYTE* key) {
    for (size_t i = 0; i < count; ++i) {
        if (std::memcmp(captured[i], key, 32) == 0) {
            return true;
        }
    }
    return false;
}

bool RestoreHook() {
    if (g_hook == nullptr) {
        return false;
    }
    std::vector<HANDLE> suspended = SuspendOtherThreads();
    DWORD oldProtect = 0;
    if (!VirtualProtect(g_hook, kPatchSize, PAGE_EXECUTE_READWRITE, &oldProtect)) {
        ResumeThreads(suspended);
        return false;
    }
    std::memcpy(g_hook, g_original, kPatchSize);
    FlushInstructionCache(GetCurrentProcess(), g_hook, kPatchSize);
    DWORD ignored = 0;
    VirtualProtect(g_hook, kPatchSize, oldProtect, &ignored);
    ResumeThreads(suspended);
    return true;
}

bool InstallKnownVersionHook(HMODULE module) {
    BYTE* base = reinterpret_cast<BYTE*>(module);
    if (!ModuleContainsRVA(module, kWeixin41155HookRva, kPatchSize) ||
        !ModuleContainsRVA(module, kWeixin41155FormatRva, 6)) {
        return false;
    }
    BYTE* hook = base + kWeixin41155HookRva;
    BYTE* format = base + kWeixin41155FormatRva;

    if (std::memcmp(format, "x'%s'\0", 6) != 0) {
        return false;
    }
    // Exact overwritten instructions for Weixin 4.1.11.55:
    //   lea rdx, [rip+format]
    //   lea rcx, [rbp+0x198]
    static constexpr BYTE secondInstruction[] = {
        0x48, 0x8D, 0x8D, 0x98, 0x01, 0x00, 0x00};
    if (hook[0] != 0x48 || hook[1] != 0x8D || hook[2] != 0x15 ||
        std::memcmp(hook + 7, secondInstruction,
                    sizeof(secondInstruction)) != 0) {
        return false;
    }
    const int32_t displacement = *reinterpret_cast<int32_t*>(hook + 3);
    if (hook + 7 + displacement != format) {
        return false;
    }

    g_capture = static_cast<CaptureSlot*>(VirtualAlloc(
        nullptr, sizeof(CaptureSlot), MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE));
    g_stub = static_cast<BYTE*>(
        VirtualAlloc(nullptr, 256, MEM_COMMIT | MEM_RESERVE, PAGE_EXECUTE_READWRITE));
    if (g_capture == nullptr || g_stub == nullptr) {
        if (g_capture != nullptr) {
            VirtualFree(g_capture, 0, MEM_RELEASE);
            g_capture = nullptr;
        }
        if (g_stub != nullptr) {
            VirtualFree(g_stub, 0, MEM_RELEASE);
            g_stub = nullptr;
        }
        return false;
    }

    BYTE* out = g_stub;
    // ENDBR64 is a no-op on older CPUs and makes the indirect detour target
    // compatible with control-flow enforcement on supported systems.
    const BYTE endbr64[] = {0xF3, 0x0F, 0x1E, 0xFA};
    EmitBytes(out, endbr64, sizeof(endbr64));
    const BYTE save[] = {
        0x9C, 0x50, 0x51, 0x52, 0x41, 0x50, 0x41, 0x51, 0x41, 0x52, 0x41, 0x53};
    EmitBytes(out, save, sizeof(save));

    // r10 = capture slot; r8 preserves the raw-key pointer from rax.
    EmitBytes(out, reinterpret_cast<const BYTE*>("\x49\xBA"), 2);
    EmitU64(out, reinterpret_cast<uint64_t>(g_capture));
    const BYTE preserveKeyPointer[] = {
        0x49, 0x89, 0xC0};  // mov r8, rax
    EmitBytes(out, preserveKeyPointer, sizeof(preserveKeyPointer));

    // Atomically claim an idle slot by changing pending from 0 to 1. Multiple
    // WeChat threads can execute this path concurrently, so a plain compare
    // followed by a store would allow overlapping copies.
    const BYTE claimSlot[] = {
        0x31, 0xC0,                          // xor eax, eax
        0x41, 0xBB, 0x01, 0x00, 0x00, 0x00,  // mov r11d, 1
        0xF0, 0x45, 0x0F, 0xB1, 0x1A,        // lock cmpxchg [r10], r11d
        0x75, 0x00};                         // jne short <after copy>
    EmitBytes(out, claimSlot, sizeof(claimSlot));
    BYTE* skipCopyDisplacement = out - 1;

    const BYTE copyKey[] = {
        0x4D, 0x8B, 0x18,              // mov r11, [r8]
        0x4D, 0x89, 0x5A, 0x08,        // mov [r10+08], r11
        0x4D, 0x8B, 0x58, 0x08,        // mov r11, [r8+08]
        0x4D, 0x89, 0x5A, 0x10,        // mov [r10+10], r11
        0x4D, 0x8B, 0x58, 0x10,        // mov r11, [r8+10]
        0x4D, 0x89, 0x5A, 0x18,        // mov [r10+18], r11
        0x4D, 0x8B, 0x58, 0x18,        // mov r11, [r8+18]
        0x4D, 0x89, 0x5A, 0x20,        // mov [r10+20], r11
        0xF0, 0x41, 0xFF, 0x02};        // lock inc dword ptr [r10] (1 -> 2)
    EmitBytes(out, copyKey, sizeof(copyKey));
    *skipCopyDisplacement = static_cast<BYTE>(out - (skipCopyDisplacement + 1));

    const BYTE restore[] = {
        0x41, 0x5B, 0x41, 0x5A, 0x41, 0x59, 0x41, 0x58,
        0x5A, 0x59, 0x58, 0x9D};
    EmitBytes(out, restore, sizeof(restore));

    // Emulate the two overwritten LEA instructions.
    EmitBytes(out, reinterpret_cast<const BYTE*>("\x48\xBA"), 2);
    EmitU64(out, reinterpret_cast<uint64_t>(format));
    const BYTE leaRcx[] = {0x48, 0x8D, 0x8D, 0x98, 0x01, 0x00, 0x00};
    EmitBytes(out, leaRcx, sizeof(leaRcx));

    // Absolute RIP-indirect jump preserves every register. Using r11 here
    // would corrupt application state because the overwritten LEAs did not.
    const BYTE jumpAbsolute[] = {0xFF, 0x25, 0x00, 0x00, 0x00, 0x00};
    EmitBytes(out, jumpAbsolute, sizeof(jumpAbsolute));
    EmitU64(out, reinterpret_cast<uint64_t>(hook + kPatchSize));

    g_hook = hook;
    std::memcpy(g_original, hook, kPatchSize);
    BYTE patch[kPatchSize] = {0xFF, 0x25, 0x00, 0x00, 0x00, 0x00};
    const uint64_t stubAddress = reinterpret_cast<uint64_t>(g_stub);
    std::memcpy(patch + 6, &stubAddress, sizeof(stubAddress));

    std::vector<HANDLE> suspended = SuspendOtherThreads();
    DWORD oldProtect = 0;
    if (!VirtualProtect(hook, kPatchSize, PAGE_EXECUTE_READWRITE, &oldProtect)) {
        ResumeThreads(suspended);
        return false;
    }
    std::memcpy(hook, patch, sizeof(patch));
    FlushInstructionCache(GetCurrentProcess(), hook, kPatchSize);
    DWORD ignored = 0;
    VirtualProtect(hook, kPatchSize, oldProtect, &ignored);
    ResumeThreads(suspended);
    return true;
}

DWORD WINAPI CaptureWorker(void*) {
    SetThreadPriority(GetCurrentThread(), THREAD_PRIORITY_HIGHEST);
    HMODULE module = nullptr;
    const ULONGLONG moduleDeadline = GetTickCount64() + 60000;
    while (module == nullptr && GetTickCount64() < moduleDeadline) {
        module = GetModuleHandleW(L"Weixin.dll");
        if (module == nullptr) {
            SwitchToThread();
        }
    }
    if (module == nullptr || !InstallKnownVersionHook(module)) {
        return 1;
    }
    SignalReadyEvent();

    BYTE captured[kMaxCapturedKeys][32] = {};
    size_t capturedCount = 0;
    const ULONGLONG captureDeadline = GetTickCount64() + 120000;
    while (GetTickCount64() < captureDeadline && capturedCount < kMaxCapturedKeys) {
        if (InterlockedCompareExchange(&g_capture->pending, kCaptureReady,
                                       kCaptureReady) != kCaptureReady) {
            Sleep(1);
            continue;
        }
        BYTE key[32] = {};
        std::memcpy(key, g_capture->key, sizeof(key));
        InterlockedExchange(&g_capture->pending, kCaptureIdle);
        if (!AlreadyCaptured(captured, capturedCount, key) && WriteCaptureFile(key)) {
            std::memcpy(captured[capturedCount], key, sizeof(key));
            ++capturedCount;
        }
        SecureZeroMemory(key, sizeof(key));
    }
    RestoreHook();
    SecureZeroMemory(captured, sizeof(captured));
    if (g_capture != nullptr) {
        SecureZeroMemory(g_capture->key, sizeof(g_capture->key));
        InterlockedExchange(&g_capture->pending, kCaptureIdle);
    }
    return capturedCount > 0 ? 0 : 2;
}

}  // namespace

BOOL APIENTRY DllMain(HMODULE module, DWORD reason, LPVOID) {
    if (reason == DLL_PROCESS_ATTACH) {
        DisableThreadLibraryCalls(module);
        HANDLE thread = CreateThread(nullptr, 0, CaptureWorker, nullptr, 0, nullptr);
        if (thread != nullptr) {
            CloseHandle(thread);
        }
    }
    return TRUE;
}
