package configs

type masterAccount struct {
	Profile *string
}

type admin struct {
	MasterAccount *masterAccount
}

// Config contains config
type Config struct {
	Admin *admin
}
