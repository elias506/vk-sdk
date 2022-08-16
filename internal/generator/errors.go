package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type SubcodeJSON map[string]float64

type ErrorJSON struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Global      *bool  `json:"global"`
	Subcodes    *[]map[string]string
}

type ErrorsFile struct {
	Definitions struct {
		Subcodes map[string]SubcodeJSON `json:"subcodes"`
	} `json:"definitions"`
	Errors map[string]ErrorJSON
}

const (
	errTypeName     = "ErrorCode"
	subcodeTypeName = "Subcode"
)

func GenerateErrors(w io.Writer, errorsRaw []byte) {
	var file ErrorsFile

	if err := json.Unmarshal(errorsRaw, &file); err != nil {
		panic(err.Error())
	}

	scs := parseSubcodes(file.Definitions.Subcodes)
	errs := parseErrors(file.Errors)

	writeStartFile(w, "vk_sdk", "// For more information about errors see https://dev.vk.com/reference/errors")

	fmt.Fprint(w, scs.Gen())
	fmt.Fprint(w, errs.Gen())
}

type Subcode struct {
	Name string
	Code int
}

func parseSubcode(name string, params SubcodeJSON) (sc Subcode) {
	sc.Name = name
	sc.Code = int(params["subcode"])
	return
}

func (sc Subcode) Gen() (gen string) {
	genName := getSubcodeName(sc.Name)

	return fmt.Sprintf("\t %s %s = %d\n", genName, subcodeTypeName, sc.Code)
}

type Subcodes []Subcode

func parseSubcodes(subcodesJSON map[string]SubcodeJSON) (scs Subcodes) {
	scs = make([]Subcode, 0, len(subcodesJSON))

	for name, sc := range subcodesJSON {
		scs = append(scs, parseSubcode(name, sc))
	}

	return
}

func (scs Subcodes) Gen() (gen string) {
	gen += fmt.Sprintf("type %s int\n\n", subcodeTypeName)

	sort.SliceStable(scs, func(i, j int) bool {
		return scs[i].Code < scs[j].Code
	})

	gen += "const (\n"

	for _, sc := range scs {
		gen += sc.Gen()
	}

	gen += ")\n\n"

	return
}

type Error struct {
	Name        string
	Code        int
	Description string
	IsGlobal    bool
	Subcodes    *[]string
	Solution    *string
}

func parseError(name string, params ErrorJSON) (e Error) {
	e.Name = name
	e.Code = params.Code
	e.Description = params.Description

	if params.Global != nil {
		e.IsGlobal = *params.Global
	}

	if params.Subcodes != nil {
		e.Subcodes = new([]string)
		*e.Subcodes = make([]string, 0, len(*params.Subcodes))

		for _, subcodeRaw := range *params.Subcodes {
			subcodeName := subcodeRaw["$ref"]

			*e.Subcodes = append(*e.Subcodes, getSubcodeName(getRefName(subcodeName)))
		}
	}

	if solution, ok := solutions[e.Code]; ok {
		e.Solution = &solution
	}

	return e
}

func (e Error) Gen() (gen string) {
	genName := getErrorName(e.Name)

	gen += fmt.Sprintf("\t// %s %s.\n", genName, e.Description)

	if e.Subcodes != nil {
		gen += fmt.Sprintf("\t// May contain one of the listed subcodes: [ %s ].\n", strings.Join(*e.Subcodes, ", "))
	}

	if e.Solution != nil {
		gen += fmt.Sprintf("\t// Solution: %s\n", *e.Solution)
	}

	gen += fmt.Sprintf("\t//  IsGlobal: %t\n", e.IsGlobal)
	gen += fmt.Sprintf("\t%s %s = %d\n", genName, errTypeName, e.Code)

	return
}

var solutions = map[int]string{
	1:   "Try again later.",
	2:   "You need to switch on the app in Settings (https://vk.com/editapp?id={Your API_ID} or use the TestMode (test_mode=1).",
	3:   "Check the method name: https://vk.com/dev/methods",
	4:   "Check if the signature has been formed correctly: https://vk.com/dev/api_nohttps.",
	5:   "Make sure that you use a correct TokenType (https://vk.com/dev/access_token).",
	6:   "Decrease the request frequency or use the execute method. More details on frequency limits here: https://vk.com/dev/api_requests.",
	7:   "Make sure that your have received required AccessPermission during the authorization (see https://vk.com/dev/permissions). You can do it with the VK.Account_GetAppPermissions method.",
	8:   "Check the request syntax (https://vk.com/dev/api_requests) and used parameters list (it can be found on a method description page).",
	9:   "You need to decrease the count of identical requests. For more efficient work you may use execute (https://vk.com/dev/execute) or JSONP (https://vk.com/dev/jsonp).",
	10:  "Try again later.",
	11:  "Switch the app off in Settings: https://vk.com/editapp?id={Your API_ID}.",
	14:  "Work with this error is explained in detail on https://vk.com/dev/captcha_error.",
	15:  "Make sure that you use correct identifiers and the content is available for the user in the full version of the site.",
	16:  "To avoid this error check if a user has the 'Use secure connection' option enabled with the VK.Account_GetInfo method.",
	17:  "Make sure that you don't use a token received with https://vk.com/dev/auth_mobile for a request from the server. It's restricted. The validation process is described on https://vk.com/dev/need_validation.",
	20:  "If you see this error despite your app has the Standalone type, make sure that you use redirect_uri=https://oauth.vk.com/blank.html. Details here: https://vk.com/dev/auth_mobile.",
	23:  "All the methods available now are listed here: https://vk.com/dev/methods.",
	24:  "Confirmation process is described on https://vk.com/dev/need_confirmation.",
	29:  "More details on rate limits here: https://vk.com/dev/data_limits",
	100: "Check the required parameters list and their format on a method description page.",
	101: "Find the app in the administrated list in settings: https://vk.com/apps?act=settings and set the correct API_ID in the request.",
	113: "Make sure that you use a correct id. You can get an id using a screen name with the VK.Utils_ResolveScreenName method",
	150: "You may get a correct value with the VK.Utils_GetServerTime method.",
	200: "Make sure you use correct ids (owner_id is always positive for users, negative for communities) and the current user has access to the requested content in the full version of the site.",
	201: "Make sure you use correct ids (owner_id is always positive for users, negative for communities) and the current user has access to the requested content in the full version of the site.",
	203: "Make sure that the current user is a member or admin of the community (for closed and private groups and events).",
	300: "You need to delete the odd objects from the album or use another album.",
	500: "Check the app settings: https://vk.com/editapp?id={Your API_ID}&section=payments",
}

type Errors []Error

func parseErrors(errorsJSON map[string]ErrorJSON) (es Errors) {
	es = make([]Error, 0, len(errorsJSON))

	for name, params := range errorsJSON {
		es = append(es, parseError(name, params))
	}

	return
}

func (es Errors) Gen() (gen string) {
	gen += fmt.Sprintf("type %s int\n\n", errTypeName)

	sort.SliceStable(es, func(i, j int) bool {
		return es[i].Code < es[j].Code
	})

	gen += "const (\n"

	for _, e := range es {
		gen += e.Gen()
	}

	gen += ")\n\n"

	return
}
