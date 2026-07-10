package ussd

import "errors"

type MenuID string

const (
	MenuMain         MenuID = "MAIN"
	MenuAirtime      MenuID = "AIRTIME"
	MenuAirtimeSelf  MenuID = "AIRTIME_SELF"
	MenuAirtimeOther MenuID = "AIRTIME_OTHER"
	MenuBundle       MenuID = "BUNDLE"
	MenuBundleSelf   MenuID = "BUNDLE_SELF"
	MenuHelp         MenuID = "HELP"
	MenuSettings     MenuID = "SETTINGS"
	MenuSendMoney    MenuID = "SEND_MONEY"
)

// Back label for all menus
const BackLabel = "Back"

type MenuEntry struct {
	Key    string // "1", "2", etc.
	Label  string // "Balance", "Airtime", etc.
	Next   MenuID // next menu, or "" if no navigation
	Action string // business action identifier, e.g. "balance", "airtime.self"
}

type Menu struct {
	ID           MenuID
	Title        string
	DisplayOrder int
	Entries      []MenuEntry
}

// MenuEngine provides menus by ID (later backed by DB/config)
type MenuEngine interface {
	GetMenu(id MenuID) (*Menu, error)
}

type staticMenuEngine struct {
	menus map[MenuID]*Menu
}

var ErrUnknownMenu = errors.New("unknown USSD menu")

func NewStaticMenuEngine() MenuEngine {
	m := &staticMenuEngine{menus: make(map[MenuID]*Menu)}

	// Main menu (USSD-first core services)
	m.menus[MenuMain] = &Menu{
		ID:           MenuMain,
		Title:        "PPay Main Menu",
		DisplayOrder: 1,
		Entries: []MenuEntry{
			{Key: "1", Label: "Balance", Action: "balance"},
			{Key: "2", Label: "Airtime", Next: MenuAirtime},
			{Key: "3", Label: "Data Bundles", Next: MenuBundle},
			{Key: "4", Label: "My Account", Action: "my_account"},
			{Key: "5", Label: "Help", Next: MenuHelp},
			{Key: "6", Label: "Settings", Next: MenuSettings},
			{Key: "7", Label: "Send Money", Next: MenuSendMoney}, // reserved
		},
	}

	// Airtime submenu
	m.menus[MenuAirtime] = &Menu{
		ID:           MenuAirtime,
		Title:        "Airtime",
		DisplayOrder: 2,
		Entries: []MenuEntry{
			{Key: "1", Label: "Self", Action: "airtime.self"},
			{Key: "2", Label: "Other Number", Action: "airtime.other"},
			{Key: "3", Label: BackLabel, Next: MenuMain},
		},
	}

	// Bundles submenu
	m.menus[MenuBundle] = &Menu{
		ID:           MenuBundle,
		Title:        "Data Bundles",
		DisplayOrder: 3,
		Entries: []MenuEntry{
			{Key: "1", Label: "Self", Action: "bundle.self"},
			{Key: "2", Label: "Other Number", Action: "bundle.other"}, // future
			{Key: "3", Label: BackLabel, Next: MenuMain},
		},
	}

	// Help submenu
	m.menus[MenuHelp] = &Menu{
		ID:           MenuHelp,
		Title:        "Help",
		DisplayOrder: 4,
		Entries: []MenuEntry{
			{Key: "1", Label: "Fees & Limits", Action: "help.fees"},
			{Key: "2", Label: "Support Contacts", Action: "help.support"},
			{Key: "3", Label: BackLabel, Next: MenuMain},
		},
	}

	// Settings submenu (reserved for future: language, notifications, etc.)
	m.menus[MenuSettings] = &Menu{
		ID:           MenuSettings,
		Title:        "Settings",
		DisplayOrder: 5,
		Entries: []MenuEntry{
			{Key: "1", Label: "Language", Action: "settings.language"},
			{Key: "2", Label: "Notifications", Action: "settings.notifications"},
			{Key: "3", Label: BackLabel, Next: MenuMain},
		},
	}

	// Send Money (reserved; can initially just show "Coming soon")
	m.menus[MenuSendMoney] = &Menu{
		ID:           MenuSendMoney,
		Title:        "Send Money",
		DisplayOrder: 6,
		Entries: []MenuEntry{
			{Key: "1", Label: "Coming soon", Action: "send_money.coming_soon"},
			{Key: "2", Label: BackLabel, Next: MenuMain},
		},
	}

	return m
}

func (m *staticMenuEngine) GetMenu(id MenuID) (*Menu, error) {
	menu, ok := m.menus[id]
	if !ok {
		return nil, ErrUnknownMenu
	}
	return menu, nil
}
