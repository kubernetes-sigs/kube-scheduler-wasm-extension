package wasm

type PluginConfig struct {
	// GuestName is the name of the guest wasm.
	GuestName string `json:"guestName"`
	// GuestPath is the path to the guest wasm.
	GuestPath string `json:"guestPath"`
}
