package config

// MetadataInitProof describes creation of a selected-account configuration,
// not successful key acquisition or an authenticated account.
type MetadataInitProof struct {
	Applied          bool   `json:"applied"`
	MetadataVerified bool   `json:"metadata_verified"`
	PrivateFile      bool   `json:"private_file"`
	SHA256           string `json:"sha256,omitempty"`
}
