package launch

// LaunchConfig holds all parameters needed to launch the game.
type LaunchConfig struct {
	WinePath         string // Used on Linux only (legacy Wine path, kept for Windows compat).
	WinePrefix       string // Used on Linux only (legacy Wine prefix, kept for Windows compat).
	ProtonScript     string // Path to the proton Python script (Linux only).
	ProtonDir        string // Root of the Proton-GE installation (Linux only).
	CompatDataPath   string // Path to Proton compatdata directory (Linux only).
	SteamInstallPath string // Detected Steam root directory (Linux only). Empty if not found.
	SteamGameId      string // Non-Steam shortcut app ID for Gamescope tracking (Linux only). "0" if not found.
	GameDir          string
	Username         string
	AccessToken      string
	OIDCTokenPath    string
	ContentBootstrap []byte
	HostX            string
	Verbose          bool
}
