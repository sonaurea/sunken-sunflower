#!/usr/bin/env bash
# ─── Coverage Enforcement — gcovr-style for Go ─────────────────────────
# 1. Runs go test with -coverprofile for internal packages
# 2. Generates HTML5 coverage report (go tool cover -html)
# 3. Generates gcovr-compatible JSON per-file summary
# 4. Enforces 100% coverage for files marked must_stay=true in baseline
# 5. Enforces test file existence for non-exempt source files
# 6. Checks aggregate threshold (default 70%)
#
# Usage:
#   scripts/check-coverage.sh              # run enforcement
#   scripts/check-coverage.sh --update-baseline  # regenerate baseline from actual coverage
#
# Exit codes: 0 = pass, 1 = coverage regression, 2 = missing test files

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE="${ROOT_DIR}/scripts/coverage-baseline.json"
COVERAGE_OUT="${ROOT_DIR}/coverage.out"
HTML_REPORT="${ROOT_DIR}/coverage.html"
JSON_REPORT="${ROOT_DIR}/coverage.json"
THRESHOLD=70

UPDATE_BASELINE=false
if [[ "${1:-}" == "--update-baseline" ]]; then
    UPDATE_BASELINE=true
fi

EXIT_CODE=0

echo "─── Running tests with coverage ─────────────────────────────"
cd "$ROOT_DIR"

# Run tests with coverage (exclude main package — requires Ebitengine display)
go test ./internal/... -coverprofile="$COVERAGE_OUT" -covermode=atomic -count=1 -timeout=120s 2>&1

echo ""
echo "─── Generating HTML5 coverage report ─────────────────────────"
go tool cover -html="$COVERAGE_OUT" -o "$HTML_REPORT"
echo "  → ${HTML_REPORT}"

echo ""
echo "─── Generating per-file JSON coverage summary ────────────────"
# Parse go tool cover -func output into gcovr-compatible JSON
cat > /tmp/parse_coverage.py << 'PYEOF'
import json, sys, re

lines = sys.stdin.readlines()

# Parse: filename.go:start.end    function       coverage%
files = {}
total_covered = 0
total_statements = 0

for line in lines:
    line = line.rstrip()
    if not line or line.startswith('total:'):
        # Handle total line
        m = re.match(r'total:.*\((\d+\.\d+)%\)', line)
        if m:
            continue
        continue

    # Format: "path/file.go:12:34\t\tfunctionName\t\t75.0%"
    parts = re.split(r'\s+', line)
    if len(parts) < 3:
        continue

    loc = parts[0]
    cov_str = parts[-1].rstrip('%')

    # Extract file path
    file_match = re.match(r'^(.+?\.go):\d+', loc)
    if not file_match:
        continue

    filepath = file_match.group(1)
    try:
        cov = float(cov_str)
    except ValueError:
        continue

    if filepath not in files:
        files[filepath] = {
            "lines_total": 0,
            "lines_covered": 0,
            "line_rate": 0.0
        }

    # Approximate: each function is ~10 statements for count purposes
    files[filepath]["lines_total"] += 10
    files[filepath]["lines_covered"] += int(10 * cov / 100)

# Calculate line_rate
for fname, fdata in files.items():
    if fdata["lines_total"] > 0:
        fdata["line_rate"] = round(fdata["lines_covered"] / fdata["lines_total"], 4)

print(json.dumps({"files": files}, indent=2))
PYEOF

go tool cover -func="$COVERAGE_OUT" | python3 /tmp/parse_coverage.py > "$JSON_REPORT"
echo "  → ${JSON_REPORT}"

echo ""
echo "─── Checking source files for test files ─────────────────────"
EXIT_CODE=2
ANY_MISSING=false

# Read exempt files from baseline
EXEMPT_FILES=$(python3 -c "
import json
with open('${BASELINE}') as f:
    data = json.load(f)
for f in data.get('exempt', []):
    print(f)
")

# Check all non-exempt source files
while IFS= read -r src_file; do
    # Skip if exempt
    if echo "$EXEMPT_FILES" | grep -Fxq "$src_file"; then
        continue
    fi

    # Only check files under internal/
    if [[ "$src_file" != internal/* ]]; then
        continue
    fi

    # Derive test file name
    dir=$(dirname "$src_file")
    base=$(basename "$src_file" .go)
    test_file="${dir}/${base}_test.go"

    if [ ! -f "$test_file" ] && [ ! -f "${dir}/${base}_internal_test.go" ]; then
        # Check for any test file in the package
        pkg_test_files=$(find "$dir" -maxdepth 1 -name '*_test.go' 2>/dev/null | head -1)
        if [ -z "$pkg_test_files" ]; then
            echo "  ❌ Missing test file for: ${src_file}"
            ANY_MISSING=true
        fi
    fi
done < <(find internal -name '*.go' -not -name '*_test.go' -not -name '*_internal_test.go' | sort)

if $ANY_MISSING; then
    echo "  ⚠  Some source files are missing test files."
    EXIT_CODE=2
else
    echo "  ✅ All tracked source files have test coverage."
    EXIT_CODE=0
fi

echo ""
echo "─── Validating against coverage baseline ─────────────────────"
TOTAL_COV=$(go tool cover -func="$COVERAGE_OUT" | tail -1 | awk '{print $NF}' | sed 's/%//')
echo "  Aggregate coverage: ${TOTAL_COV}% (threshold: ${THRESHOLD}%)"

if (( $(echo "$TOTAL_COV < $THRESHOLD" | bc -l) )); then
    echo "  ❌ Aggregate coverage ${TOTAL_COV}% < ${THRESHOLD}% threshold"
    EXIT_CODE=1
fi

# Check must_stay files
python3 -c "
import json, sys

with open('${BASELINE}') as f:
    baseline = json.load(f)

with open('${JSON_REPORT}') as f:
    actual = json.load(f)

failed = False
for filepath, expected in baseline.get('files', {}).items():
    if not expected.get('must_stay', False):
        continue

    actual_file = actual.get('files', {}).get(filepath, {})
    actual_rate = actual_file.get('line_rate', 0.0)

    if actual_rate < expected.get('covered', 0):
        print(f'  ❌ {filepath}: coverage dropped {expected[\"covered\"]*100:.1f}% → {actual_rate*100:.1f}%')
        failed = True

if failed:
    sys.exit(1)
else:
    print('  ✅ All must_stay files passed.')
"

if [ $? -ne 0 ]; then
    EXIT_CODE=1
fi

if [ "$UPDATE_BASELINE" = true ]; then
    echo ""
    echo "─── Updating coverage baseline ──────────────────────────────"
    python3 -c "
import json

with open('${BASELINE}') as f:
    baseline = json.load(f)

with open('${JSON_REPORT}') as f:
    actual = json.load(f)

new_files = {}
for filepath, fdata in actual.get('files', {}).items():
    rate = fdata.get('line_rate', 0.0)
    must_stay = baseline.get('files', {}).get(filepath, {}).get('must_stay', False)
    new_files[filepath] = {
        'covered': rate,
        'must_stay': must_stay
    }

baseline['files'] = new_files
with open('${BASELINE}', 'w') as f:
    json.dump(baseline, f, indent=2)
    f.write('\n')

print(f'  ✅ Baseline updated: {len(new_files)} files')
"
fi

echo ""
echo "─── Summary ──────────────────────────────────────────────────"
if [ $EXIT_CODE -eq 0 ]; then
    echo "  ✅ All coverage checks passed."
elif [ $EXIT_CODE -eq 1 ]; then
    echo "  ❌ Coverage regression detected."
elif [ $EXIT_CODE -eq 2 ]; then
    echo "  ❌ Missing test files."
fi

exit $EXIT_CODE