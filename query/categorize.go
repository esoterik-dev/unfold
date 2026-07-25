package query
import (
	"strings"
	"unicode"
)

// MERCHANT_OVERRIDES maps uppercase merchant names to cleaned display names.
var MERCHANT_OVERRIDES = map[string]string{
	"SWIGGY": "Swiggy", "ZOMATO": "Zomato", "BLINKIT": "Blinkit",
	"ZEPTO": "Zepto", "BIGBASKET": "BigBasket", "DUNZO": "Dunzo",
	"AMAZON": "Amazon", "FLIPKART": "Flipkart", "MYNTRA": "Myntra",
	"AJIO": "AJIO", "NYKAA": "Nykaa", "MEESHO": "Meesho",
	"NETFLIX": "Netflix", "SPOTIFY": "Spotify", "YOUTUBE": "YouTube",
	"HOTSTAR": "Hotstar", "BOOKMYSHOW": "BookMyShow",
	"UBER": "Uber", "OLA": "Ola", "RAPIDO": "Rapido",
	"PHONEPE": "PhonePe", "PAYTM": "Paytm", "GPAY": "Google Pay",
	"GOOGLE PAY": "Google Pay", "CRED": "CRED",
	"ZERODHA": "Zerodha", "GROWW": "Groww", "UPSTOX": "Upstox",
	"HDFC BANK": "HDFC Bank", "ICICI BANK": "ICICI Bank",
	"AXIS BANK": "Axis Bank", "SBI YONO": "SBI (YONO)",
	"KOTAK MAHINDRA": "Kotak Bank", "INDUSIND BANK": "IndusInd Bank",
	"AIRTEL": "Airtel", "JIO": "Jio", "VODAFONE": "Vodafone",
	"IRCTC": "IRCTC", "MAKEMYTRIP": "MakeMyTrip", "REDBUS": "redBus",
}

// CATEGORY_ID_MAP maps Fold's UUID category IDs to human-readable names.
var CATEGORY_ID_MAP = map[string]string{
	"07ef4989-6002-4fe2-98d8-c2a6429904e8": "Health & Fitness",
	"08d63a76-6e11-4467-854e-5a081fd51097": "Groceries",
	"0b55293c-5b8d-48e4-91be-4312b87dd714": "Food & Drinks",
	"247e3e5d-59bf-4bf8-82ff-c152569893ea": "Self Transfer",
	"255b80bb-96c7-4254-be68-8b80dd1fc473": "Refunds",
	"2ca04e4b-45e8-4e06-8029-9b1eeb35fc44": "Medical",
	"2f20c891-b2d4-4a1b-b3aa-d02c0deffb02": "Loans / EMIs",
	"3cf94230-f5bf-4aa7-a3f0-bb0c4453413f": "Transfer / People",
	"3d3a06cc-92cb-4d88-b9c9-4ce53de25c17": "Transfer / People",
	"46a9a71e-388e-418e-a13b-cc7d140f29b8": "Investments",
	"46ab2657-0e00-40cf-af5d-5d65e4e58152": "Transfer / People",
	"478baaf7-fe5d-4f96-ba96-f5e570782886": "Donations",
	"54b06ae4-b19e-402e-9940-af8b2fec6a3d": "Investments",
	"6176feae-21c5-4874-9d9a-908a257e0ede": "Transfer / People",
	"6c41313d-82ed-4333-9329-ac75551e79c5": "Shopping",
	"6e7a92ec-2798-408e-b164-6283d63a1ca9": "Entertainment",
	"7505b993-d7fc-4461-9dc9-85c027a367ae": "Transfer / People",
	"80b59092-aa0b-4b55-b136-4e7c8c68a31f": "Credit Card Bill",
	"8281a1f2-df95-4299-97cc-c93b1e0a65ee": "Transfer / People",
	"a1ff63ba-1712-4753-a558-2a051c94e709": "Subscriptions",
	"a4303fd3-a231-44fe-aae3-47561855689e": "Shopping",
	"a4ec128f-b01f-4a86-98f2-156c8b2b6071": "Investments",
	"a9c18a13-2b0a-4006-847a-83e16045870c": "Self Transfer",
	"b3f7d021-7853-4ebd-836c-769f3f5539f0": "Transfer / People",
	"cbfad38c-afc8-49d3-b689-7171767df0d6": "Salary",
	"e5f0f35e-716e-4557-a10f-fc6c18c253c4": "Bills & Utilities",
	"e893b723-228f-4732-8c54-929f967e9364": "Interest / Returns",
	"ea8d1a84-e017-4d28-8f98-9e2b0f752936": "Transport & Services",
	"eeec0a99-9940-4361-92f6-12852fc8a051": "Bank Charges",
	"f181114a-e486-4860-acc5-d95822a9d73f": "Transport & Fuel",
	"f22d1ec8-5cc7-4b4e-b0d3-4193e757f52d": "Cashback / Rewards",
	"f26aa519-14ba-47dc-a07e-7a6ff7a8dc1f": "Entertainment",
	"f2d5dcf1-0c54-4f8d-957a-ae9b63e458f1": "Transfer / People",
	"f48dec4d-6f58-44ae-b6df-c34dfc1d6c83": "Transfer / People",
}

// SMART_OVERRIDES maps cleaned lowercase merchant names to human categories.
var SMART_OVERRIDES = map[string]string{
	"swiggy":                           "Food Delivery",
	"zomato":                           "Food Delivery",
	"blinkit":                          "Groceries",
	"zepto":                            "Groceries",
	"bigbasket":                        "Groceries",
	"blinkit commerce private limited": "Groceries",
	"freshco":                          "Groceries",
	"meat mart":                        "Groceries",
	"bmtc bus":                         "Public Transport",
	"rapido":                           "Cabs & Autos",
	"uber":                             "Cabs & Autos",
	"autorickshaw":                     "Cabs & Autos",
	"ksrtc ban":                        "Travel",
	"amazon":                           "Online Shopping",
	"flipkart":                         "Online Shopping",
	"ikea":                             "Furniture & Home",
	"decathlon":                        "Sports & Fitness",
	"bookmyshow":                       "Entertainment",
	"youtube":                          "Subscriptions",
	"apple":                            "Subscriptions",
	"apple services":                   "Subscriptions",
	"rentomojo":                        "Rent",
	"reliance jio":                     "Telecom",
	"airtel payments bank":             "Telecom",
	"lazypay":                          "Credit Card Bill",
	"gokiwi tech private":              "Credit Card Bill",
	"groww":                            "Investments",
	"indian clearing corporation limited": "Investments",
	"canara bank":                      "Self Transfer",
	"hdfc bank":                        "Self Transfer",
}

// CATEGORY_KEYWORDS_FALLBACK is a list of keyword-based category fallbacks.
var CATEGORY_KEYWORDS_FALLBACK = []struct {
	Category string
	Keywords []string
}{
	{"Food Delivery", []string{"swiggy", "zomato"}},
	{"Quick Commerce", []string{"blinkit", "zepto", "dunzo", "instamart", "bigbasket", "grofers"}},
	{"Transport", []string{"uber", "ola", "rapido", "metro"}},
	{"Travel", []string{"irctc", "redbus", "makemytrip", "goibibo", "yatra", "cleartrip", "indigo", "spicejet", "air india", "easemytrip", "airasia", "vistara"}},
	{"Entertainment", []string{"netflix", "hotstar", "amazon prime", "spotify", "youtube", "pvr", "inox", "bookmyshow", "disney", "jiocinema"}},
	{"Telecom", []string{"airtel", "jio", "vodafone", " vi ", "bsnl", "act broadband"}},
	{"Shopping", []string{"amazon", "flipkart", "myntra", "ajio", "nykaa", "meesho", "tatacliq", "snapdeal"}},
	{"Health & Fitness", []string{"apollo", "1mg", "practo", "netmeds", "medplus", "cult.fit", "gym"}},
	{"Investing", []string{"zerodha", "groww", "upstox", "smallcase", "kuvera", "paytm money"}},
	{"Education", []string{"udemy", "coursera", "unacademy", "byju", "vedantu", "duolingo"}},
	{"Fuel", []string{"petrol", "bpcl", "hpcl", "iocl", "nayara"}},
	{"Utilities", []string{"electricity", "water bill", "bescom", "tata power", "adani electricity", "mahanagar gas"}},
}

// CleanMerchant normalizes a raw merchant name from the API.
func CleanMerchant(name string) string {
	if name == "" {
		return name
	}
	trimmed := strings.TrimSpace(name)
	upper := strings.ToUpper(trimmed)

	// Exact override
	if val, ok := MERCHANT_OVERRIDES[upper]; ok {
		return val
	}

	// Prefix override (e.g. "SWIGGY INSTAMART" -> "Swiggy Instamart")
	for key, val := range MERCHANT_OVERRIDES {
		if strings.HasPrefix(upper, key+" ") || strings.HasPrefix(upper, key+"_") {
			suffix := trimmed[len(key):]
			cleanSuffix := toTitleCase(suffix)
			return val + cleanSuffix
		}
	}

	// Strip legal suffixes
	out := trimmed
	for _, suffix := range []string{" PRIVATE LIMITED", " PVT. LTD.", " PVT LTD", " LIMITED"} {
		if strings.HasSuffix(strings.ToUpper(out), suffix) {
			out = strings.TrimSpace(out[:len(out)-len(suffix)])
		}
	}

	// Title-case if still entirely uppercase and longer than 3 chars
	if len(out) > 3 && isAllUpper(out) {
		out = toTitleCase(out)
	}

	return out
}

// toTitleCase capitalizes the first letter of each word.
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// isAllUpper checks if all letters in the string are uppercase or non-letters.
func isAllUpper(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

// Categorize maps a Fold category UUID and merchant name to a human-readable category.
func Categorize(categoryID, merchant string) string {
	clean := strings.ToLower(CleanMerchant(merchant))

	// 1. Smart Overrides (exact match)
	if val, ok := SMART_OVERRIDES[clean]; ok {
		return val
	}

	// 2. Smart Overrides (partial match)
	for key, val := range SMART_OVERRIDES {
		if strings.Contains(clean, key) {
			return val
		}
	}

	// 3. Fold UUID Native Category
	if categoryID != "" {
		if val, ok := CATEGORY_ID_MAP[categoryID]; ok {
			return val
		}
	}

	// 4. Legacy Keyword Fallback
	for _, entry := range CATEGORY_KEYWORDS_FALLBACK {
		for _, kw := range entry.Keywords {
			if strings.Contains(clean, kw) {
				return entry.Category
			}
		}
	}

	return "Other"
}

// Truncate shortens a string to max characters with an ellipsis.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "…"
}
