package wsl

// HostVersion is the trellis-cli version of the Windows host binary. It is set
// from main at startup (mirroring main.version) and used to keep the in-distro
// binary in sync with the host. Defaults to "canary" for local dev builds.
var HostVersion = "canary"
