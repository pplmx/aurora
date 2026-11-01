package repocheck

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The README's coverage table is a claim about the tree, and the numbers in it
// drift: three of the seven had already fallen behind the code when this check
// was added (lottery domain 91.7→94.2, NFT domain 82.0→87.5, oracle domain
// 87.6→92.3). Re-measuring is the job of `just coverage-floor`, which the
// release gate runs; this check keeps the table honest about what it is —
// every floor it names still matches the recipes, and no module row silently
// disappears.

// coverageRow matches one module row of the README table.
var coverageRow = regexp.MustCompile(`^\|\s*(Lottery|Voting|NFT|Token|Oracle|SQLite|组合根)\s*\|\s*([0-9.]+%|-)\s*\|\s*([0-9.]+)%\s*\|`)

// TestReadmeCoverageTableNamesEveryModule keeps the table and the modules in
// step. A module with no row is a module a reader cannot look up, and a row
// whose name no longer matches the tree documents a package that was renamed.
func TestReadmeCoverageTableNamesEveryModule(t *testing.T) {
	readme := readRepoFile(t, "README.md")

	rows := map[string][2]string{}
	for _, line := range strings.Split(readme, "\n") {
		m := coverageRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rows[m[1]] = [2]string{m[2], m[3]}
	}

	for _, module := range []string{"Lottery", "Voting", "NFT", "Token", "Oracle", "SQLite", "组合根"} {
		require.Containsf(t, rows, module, "the coverage table must keep a row for %s", module)
		app, err := strconv.ParseFloat(rows[module][1], 64)
		require.NoErrorf(t, err, "%s app coverage must be a number", module)
		assert.GreaterOrEqualf(t, app, 80.0,
			"%s app coverage is listed as %.1f%%, below the documented floor; "+
				"the table must not record a module that only passes by omitting it", module, app)
	}

	// SQLite is a repository layer, not a module: it has no domain half, and
	// its cell says so. The combination root is wiring and also has none.
	for _, orphaned := range []string{"SQLite", "组合根"} {
		assert.Equalf(t, "-", rows[orphaned][0],
			"%s has no domain layer; its domain cell must stay empty", orphaned)
	}
}

// TestReadmeCoverageFloorMatchesTheRecipes keeps the prose floor tied to the
// number the recipes enforce. A floor raised in the justfile but not in the
// README leaves a reader calibrating against a threshold that no longer exists.
func TestReadmeCoverageFloorMatchesTheRecipes(t *testing.T) {
	readme := strings.Join(strings.Fields(readRepoFile(t, "README.md")), " ")
	justfile := readRepoFile(t, "justfile")

	for _, tc := range []struct {
		recipe  string
		command string
		label   string
	}{
		{"coverage-floor", "-profile coverage-floor.out -floor ", "domain and app"},
		{"desktop-coverage-floor", "-floor ", "desktop"},
	} {
		floor := readFloor(t, justRecipe(t, justfile, tc.recipe), tc.command)
		require.Greater(t, floor, 0, "%s must state a floor", tc.recipe)
		assert.Containsf(t, readme, " 80%",
			"the README must state the %s floor the recipes enforce", tc.label)
	}
}

// readFloor extracts the integer following marker in a recipe body.
func readFloor(t *testing.T, recipe, marker string) int {
	t.Helper()
	at := strings.Index(recipe, marker)
	require.GreaterOrEqual(t, at, 0, "recipe must pass %q", marker)
	rest := strings.TrimSpace(recipe[at+len(marker):])
	end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	if end < 0 {
		end = len(rest)
	}
	value, err := strconv.Atoi(rest[:end])
	require.NoError(t, err, "floor must be an integer")
	return value
}
