package generator

import (
	"fmt"
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

type MovieDoc struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Synopsis       string   `json:"synopsis"`
	Genre          string   `json:"genre"`
	ReleaseYear    int      `json:"release_year"`
	Director       string   `json:"director"`
	Cast           []string `json:"cast"`
	Language       string   `json:"language"`
	RuntimeMinutes int      `json:"runtime_minutes"`
	Rating         float64  `json:"rating"`
}

type BookDoc struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	Author          string  `json:"author"`
	Genre           string  `json:"genre"`
	ISBN            string  `json:"isbn"`
	PublicationYear int     `json:"publication_year"`
	Language        string  `json:"language"`
	PageCount       int     `json:"page_count"`
	Rating          float64 `json:"rating"`
}

var (
	languages = []string{
		"English", "Japanese", "Korean", "Spanish", "French", "Hindi", "Italian",
	}
	directorNames = []string{
		"Maya Hart", "Daniel Okafor", "Lena Fujimoto", "Marco Alvarez", "Asha Raman",
		"Elena Petrov", "Noah Bennett", "Sofia Laurent", "Jun Park", "Nadia El-Sayed",
		"Gabriel Costa", "Priya Menon", "Tommaso Ricci", "Claire Moreau", "Mateo Silva",
	}
	actorNames = []string{
		"Aria Collins", "Theo Nakamura", "Ines Navarro", "Ravi Patel", "Camila Duarte",
		"Jonas Weber", "Leila Haddad", "Mina Sato", "Elias Brooks", "Sana Rahman",
		"Omar Khalid", "Nora Bell", "Diego Mendes", "Aiko Tanaka", "Julian Cross",
		"Meera Kapoor", "Lucia Ferraro", "Kian Morgan", "Yara Hussein", "Evan Mercer",
	}
	authorNames = []string{
		"Clara Winthrop", "Adrian Vale", "Naomi Mercer", "Tariq Hassan", "Isabella Shore",
		"Kenji Morita", "Leonie Bauer", "Marta Kovacs", "Rohan Sen", "Selene Armitage",
		"Eva Sinclair", "Luca Marino", "Hana Idris", "Mateo Quinn", "Yuki Sakamoto",
	}
	movieTitleAdjectives = []string{
		"Silent", "Last", "Hidden", "Broken", "Crimson", "Golden", "Shattered", "Burning",
		"Fading", "Midnight", "Vanishing", "Electric", "Secret", "Lonely", "Fierce",
	}
	movieTitleNouns = []string{
		"Harbor", "Meridian", "Signal", "Ember", "Frontier", "Archive", "Witness", "Storm",
		"Passage", "Orbit", "Reckoning", "Labyrinth", "Promise", "Kingdom", "River",
	}
	movieTitleQualifiers = []string{
		"North", "Glass", "Silver", "Fifth", "Velvet", "Hollow", "Neon", "Winter",
	}
	movieProtagonists = []string{
		"a grieving translator", "an ambitious detective", "a washed-up stunt pilot",
		"a young composer", "a disillusioned journalist", "an exiled cartographer",
		"a rookie surgeon", "a veteran thief", "a reluctant prince", "an undercover archivist",
	}
	movieSettings = []string{
		"in a floodlit port city", "on a remote lunar outpost", "inside a decaying mountain resort",
		"across a windswept archipelago", "within a glittering megacity", "in a drought-stricken valley",
		"inside a secluded monastery", "through a war-scarred border town", "in a lavish coastal estate",
	}
	movieConflicts = map[string][]string{
		"Science Fiction": {"must decode a signal that predicts political assassinations", "finds a machine that rewrites human memory"},
		"Fantasy":         {"awakens an oath-bound creature beneath the city", "inherits a map that opens hidden kingdoms"},
		"Thriller":        {"uncovers a blackmail network tied to a vanished judge", "becomes the only witness to a staged disappearance"},
		"Drama":           {"returns home to settle a family debt that reopened old betrayals", "must choose between public loyalty and private truth"},
		"Mystery":         {"investigates a locked-room death no one wants solved", "follows a trail of letters left by a missing historian"},
		"Crime":           {"is drawn into a jewel heist planned inside a diplomatic summit", "must broker peace between rival crews before a truce collapses"},
		"Adventure":       {"leads an expedition after a lost convoy resurfaces on old film reels", "crosses hostile terrain to recover a stolen relic"},
		"Animation":       {"befriends a runaway inventor whose machines can reshape weather", "guides a troupe of dream creatures back to their vanished forest"},
		"Romance":         {"meets a former rival while restoring a theater scheduled for demolition", "finds love while hiding an identity that could upend two families"},
	}
	movieStakes = []string{
		"before the city descends into panic",
		"before a fragile peace agreement collapses",
		"before a powerful family buries the truth",
		"before the final ferry leaves for the season",
		"before the only evidence disappears into private hands",
		"before the storm sealing the valley turns permanent",
	}
	bookTitleNouns = []string{
		"Lantern", "Atlas", "Garden", "Harbor", "Cathedral", "Map", "Winter", "Tide",
		"Library", "Province", "House", "Mirror", "Archive", "Portrait", "Silence",
	}
	bookTitleQualifiers = []string{
		"Glass", "Blackwater", "Northern", "Seventh", "Ashen", "Ivory", "Hidden", "Amber",
	}
	bookProtagonists = []string{
		"a reluctant archivist", "a disgraced botanist", "a widowed astronomer",
		"an idealistic magistrate", "a sharp-tongued governess", "an orphaned mapmaker",
		"a battlefield nurse", "a debt-ridden bookseller", "a novice monk", "a restless violinist",
	}
	bookSettings = []string{
		"in a rain-soaked capital", "within a remote observatory", "inside a declining manor house",
		"across a disputed frontier", "inside a desert university", "along a frozen trade route",
		"within a cloistered island town", "inside a court of elaborate etiquette",
	}
	bookConflicts = map[string][]string{
		"Fantasy":            {"inherits a key that can wake a sleeping citadel", "must bargain with a river spirit to save a famine-struck province"},
		"Science Fiction":    {"finds a journal describing a colony that officially never existed", "must choose whether to expose an experiment that edits grief"},
		"Mystery":            {"investigates the death of a patron whose will vanished overnight", "discovers a pattern linking a string of thefts to an abandoned chapel"},
		"Historical Fiction": {"is pulled into a conspiracy surrounding a royal succession", "must preserve forbidden letters that could rewrite a war's legacy"},
		"Literary Fiction":   {"returns home after a decade abroad to confront a family silence", "tries to rebuild a life while caring for an estranged parent"},
		"Romance":            {"forms an uneasy alliance with a rival scholar during a winter residency", "falls for a negotiator whose loyalties lie across the border"},
		"Thriller":           {"uncovers a smuggling route hidden inside library shipments", "becomes the target of a financier determined to reclaim a coded ledger"},
		"Adventure":          {"joins an overland caravan searching for a vanished city", "must outrun mercenaries to deliver a sealed astrolabe"},
		"Classic":            {"navigates inheritance, reputation, and social ambition in a watchful town", "learns how gossip and duty can alter the fate of an entire household"},
	}
	bookStakes = []string{
		"before the next council session decides the region's future",
		"before winter closes the passes",
		"before the estate is sold to settle a debt",
		"before the archive is sealed by the crown",
		"before the evidence is erased by the very people funding the search",
	}
)

func nextID(prefix string, counter *atomic.Uint64) string {
	return fmt.Sprintf("%s-%d-%06d", prefix, time.Now().UTC().UnixNano(), counter.Add(1))
}

func pick[T any](rng *rand.Rand, items []T) T {
	return items[rng.Intn(len(items))]
}

func pickDistinct(rng *rand.Rand, items []string, count int) []string {
	if count >= len(items) {
		cloned := append([]string(nil), items...)
		rng.Shuffle(len(cloned), func(i, j int) {
			cloned[i], cloned[j] = cloned[j], cloned[i]
		})
		return cloned
	}

	selected := make(map[string]struct{}, count)
	result := make([]string, 0, count)
	for len(result) < count {
		item := pick(rng, items)
		if _, exists := selected[item]; exists {
			continue
		}
		selected[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func weightedYear(rng *rand.Rand, minYear int, maxYear int) int {
	rangeSize := maxYear - minYear + 1
	base := minYear + rng.Intn(rangeSize)
	recentBoost := rng.Intn(rangeSize / 3)
	year := base + recentBoost
	if year > maxYear {
		year = maxYear - rng.Intn(3)
	}
	return year
}

func ratingBetween(rng *rand.Rand, minValue float64, maxValue float64) float64 {
	value := minValue + rng.Float64()*(maxValue-minValue)
	return math.Round(value*10) / 10
}

func randomInt(rng *rand.Rand, minValue int, maxValue int) int {
	return minValue + rng.Intn(maxValue-minValue+1)
}

func checksumISBN13(prefix string) string {
	sum := 0
	for idx, r := range prefix {
		digit := int(r - '0')
		if idx%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}
	checkDigit := (10 - (sum % 10)) % 10
	return fmt.Sprintf("%s%d", prefix, checkDigit)
}
