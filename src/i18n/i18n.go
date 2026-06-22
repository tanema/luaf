package i18n

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

type (
	// Locale captures a single local setting, usually the current system locale.
	Locale struct {
		tag language.Tag
	}
	// LocaleCategory is an enum of which category to set the locale setting on.
	LocaleCategory string
)

const (
	// CategoryALL sets all categories.
	CategoryALL LocaleCategory = "all"
	// CategoryCOLLATE sets collation, regex and collation functions.
	CategoryCOLLATE LocaleCategory = "collate"
	// CategoryCTYPE sets char classification, regex, char conversions, wide-char functions.
	CategoryCTYPE LocaleCategory = "ctype"
	// CategoryMESSAGES sets the locales on system messages.
	CategoryMESSAGES LocaleCategory = "messages"
	// CategoryMONETARY sets monetary formatting.
	CategoryMONETARY LocaleCategory = "monetary"
	// CategoryNUMERIC sets numeric formatting.
	CategoryNUMERIC LocaleCategory = "numeric"
	// CategoryTIME sets time formatting.
	CategoryTIME LocaleCategory = "time"
)

var (
	supported = []language.Tag{
		language.English, // en
	}
	matcher = language.NewMatcher(supported)
	// DefaultLocale is the locale that the VM starts with. In the future this should
	// probably retrieved from the system rather than just set to english.
	DefaultLocale = &Locale{tag: language.English}
	currentLocale = DefaultLocale
)

// ParseLocale will parse a string to establish a locale. If the string is not a valid
// locale, it will return an error. If the language is not supported then it will
// return an error. If the local is valid and supported, it will return the locale.
func ParseLocale(str string) (*Locale, error) {
	t, _, err := language.ParseAcceptLanguage(str)
	if err != nil {
		return nil, err
	}
	tag, idx, _ := matcher.Match(t...)
	if idx == -1 {
		return nil, fmt.Errorf("language %s is not supported currently", str)
	}
	return &Locale{tag: tag}, nil
}

// ParseCategory will parse a string to match it to a LocaleCategory. It will
// return a ParseCategory if it is correct or an error if the category is incorrect.
func ParseCategory(cat string) (LocaleCategory, error) {
	switch strings.ToLower(cat) {
	case string(CategoryALL):
		return CategoryALL, nil
	case string(CategoryCOLLATE):
		return CategoryCOLLATE, nil
	case string(CategoryCTYPE):
		return CategoryCTYPE, nil
	case string(CategoryMESSAGES):
		return CategoryMESSAGES, nil
	case string(CategoryMONETARY):
		return CategoryMONETARY, nil
	case string(CategoryNUMERIC):
		return CategoryNUMERIC, nil
	case string(CategoryTIME):
		return CategoryTIME, nil
	default:
		return "", fmt.Errorf("unknown local category %s", cat)
	}
}

// SetLocale will set the running processes locale.
func SetLocale(locale *Locale, _ LocaleCategory) {
	// cat not used yet
	currentLocale = locale
}

// GetLocale will return the local currently set for the category.
func GetLocale(_ LocaleCategory) *Locale {
	// cat not used yet
	return currentLocale
}

// String will return a human readable string for the locale.
func (locale *Locale) String() string {
	return locale.tag.String()
}
