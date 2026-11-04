package pkgid

const AppID = "App"

func Packaged() bool { return FamilyName() != "" }

func AUMID() string {
	if f := FamilyName(); f != "" {
		return f + "!" + AppID
	}
	return ""
}
