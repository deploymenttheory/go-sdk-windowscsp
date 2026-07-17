package naming

import (
	"reflect"
	"testing"
)

func TestPackageName(t *testing.T) {
	cases := map[string]string{
		"LAPS":                    "laps",
		"Camera_AreaDDF":          "camera",
		"ADMX_AppCompat_AreaDDF":  "admx_appcompat",
		"ADMX_MSS-legacy_AreaDDF": "admx_mss_legacy",
		"WiFi":                    "wifi",
		"eUICCs":                  "euiccs",
		"EMAIL2":                  "email2",
		"":                        "csp",
		"3G":                      "csp3g",
	}
	for in, want := range cases {
		if got := PackageName(in); got != want {
			t.Errorf("PackageName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExportName(t *testing.T) {
	cases := map[string]string{
		"AllowCamera":   "AllowCamera",
		"ADMX_Bits":     "ADMXBits",
		"profile-name":  "ProfileName",
		"3GPP":          "N3GPP",
		"MSS-legacy":    "MSSLegacy",
		"":              "X",
		"lower case ok": "LowerCaseOk",
	}
	for in, want := range cases {
		if got := ExportName(in); got != want {
			t.Errorf("ExportName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinExport(t *testing.T) {
	if got := JoinExport([]string{"Schedule", "DailyRecurrent"}); got != "ScheduleDailyRecurrent" {
		t.Errorf("JoinExport = %q", got)
	}
	if got := JoinExport(nil); got != "Root" {
		t.Errorf("JoinExport(nil) = %q", got)
	}
}

func TestParamName(t *testing.T) {
	cases := map[string]string{
		"ProfileName": "profileName",
		"LocURI":      "locURI",
		"Item":        "item",
		"type":        "typeName", // reserved
		"value":       "valueName",
	}
	for in, want := range cases {
		if got := ParamName(in); got != want {
			t.Errorf("ParamName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConstName(t *testing.T) {
	cases := []struct{ desc, value, want string }{
		{"Not allowed.", "0", "NotAllowed"},
		{"Allowed.", "1", "Allowed"},
		{"", "42", "Value42"},
		{"Block all downloads (recommended). Extra text ignored.", "2", "BlockAllDownloads"},
		{"One two three four five six seven eight", "3", "OneTwoThreeFourFiveSix"},
	}
	for _, c := range cases {
		if got := ConstName(c.desc, c.value); got != c.want {
			t.Errorf("ConstName(%q, %q) = %q, want %q", c.desc, c.value, got, c.want)
		}
	}
}

func TestWrapComment(t *testing.T) {
	got := WrapComment("aaa bbb ccc ddd", 7)
	want := []string{"aaa bbb", "ccc ddd"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WrapComment = %v, want %v", got, want)
	}
	if WrapComment("", 10) != nil {
		t.Error("WrapComment(empty) should be nil")
	}
}

func TestImportAlias(t *testing.T) {
	if got := ImportAlias("policy", "admx_appcompat"); got != "policyadmxappcompat" {
		t.Errorf("ImportAlias = %q", got)
	}
}
