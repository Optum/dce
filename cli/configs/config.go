package configs

// Auth contains auth config
type masterAccount struct {
	Profile *string
}

// Auth contains auth config
type admin struct {
	MasterAccount *masterAccount
}

// Config contains config
type Config struct {
	Admin *admin
}
