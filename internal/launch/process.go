package launch

// LaunchConfig holds all parameters needed to launch the game.
type LaunchConfig struct {
	WinePath         string // Used on Linux only.
	WinePrefix       string // Used on Linux only.
	GameDir          string
	Username         string
	AccessToken      string
	OIDCTokenPath    string
	ContentBootstrap []byte
	HostX            string
	Verbose          bool
}
