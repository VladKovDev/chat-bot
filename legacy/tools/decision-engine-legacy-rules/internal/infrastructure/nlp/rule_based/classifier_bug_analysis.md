# NLP Rule-Based Classifier - Bug Analysis Report

## Test Results Summary

### ✅ Working Features
- Single keyword matching works
- Multiple keywords from same intent accumulate score
- Exact phrase matching works (when tokens match exactly)

### ❌ Identified Bugs

## Bug #1: Phrase Matching with Stopwords **CRITICAL**

**Test Case:**
```go
tokens: ["связаться", "с", "оператор"]  // Exact match
phrase in config: "связаться с оператором"
Expected: Should match
Result: unknown ❌
```

**Root Cause:**
The classifier compiles phrases using `strings.Fields()` which splits on whitespace:
- Config phrase: `"связаться с оператором"` → `["связаться", "с", "оператором"]`
- Input tokens: `["связаться", "с", "оператор"]`
- Comparison: `"оператор" != "оператором"` ❌

**Problem:** Input tokens are lemmatized (оператор), but config phrases store original forms (оператором).

---

## Bug #2: Confidence Calculation **CRITICAL**

**Test Case:**
```
Intent: request_operator
- 6 keywords × 1.5 weight = 9.0
- 3 phrases × 5.0 weight = 15.0
- maxPossible = 24.0

Input: ["оператор", "человек", "бот", "робот"]  (4 matches!)
Score: 6.0
Confidence: 6.0 / 24.0 = 0.250
Threshold: 0.3
Result: Below threshold, should NOT match ❌
```

**Root Cause:**
`maxPossible` is calculated as sum of ALL rule weights:
```go
r.maxPossible[intent.Name] += rule.Weight * len(rule.Values)
```

But this is WRONG because:
1. Each input token can only match ONE rule (due to `seen` map in classifier)
2. User can never provide all possible keywords + all possible phrases in one message
3. Confidence is severely underestimated

**Example:**
- Intent "request_operator" has 18 keywords + 27 phrases
- User says: "оператор" (1 keyword match)
- Score: 1.5, maxPossible: ~163.5
- Confidence: 0.009 (virtually zero!)

---

## Bug #3: Ambiguity Delta

**Similar issue to Bug #2:**
```go
if best.score-second.score < r.cfg.Threshold.AmbiguityDelta*max {
```

The delta is multiplied by `maxPossible`, which is artificially large.
With maxPossible = 163.5 and delta = 0.15:
- Required difference: 0.15 × 163.5 = 24.5 points
- But typical phrase weight is only 5.0!
- Ambiguity detection will NEVER trigger.

---

## Bug #4: Rules.json Configuration Issues

**Problem:** Phrases in config have prepositions, but normalization removes them.

```json
{
  "type": "phrase",
  "values": ["связаться с оператором", "поговорить с человеком"]
}
```

After normalization (stopwords removal):
- Input: `"связаться с оператором"` → `["связаться", "оператор"]`
- Config phrase: `["связаться", "с", "оператором"]`
- Mismatch: 2 tokens vs 3 tokens ❌

**Additional Issues:**
- Phrases have different morphological forms (оператор, оператору, оператором)
- No lemmatization during config compilation
- MWE (multi-word expressions) only handles "не_" + word, not other multi-word phrases

---

## Impact Assessment

### Current State (with thresholds = 0)
- ✅ Single keywords work
- ✅ Multiple keywords accumulate
- ❌ Phrases DON'T work (lemmatization mismatch)
- ⚠️ Confidence is meaningless (always ~0.01-0.05)
- ⚠️ Ambiguity detection is broken
- ⚠️ Thresholds must be set to 0 to get any results

### Why It "Works" with threshold = 0
The classifier returns results despite low confidence because:
1. Threshold check is bypassed (min_confidence = 0)
2. Best matching intent is returned regardless of confidence
3. No ambiguity warnings (delta check is broken)

---

## Recommended Solutions

### Solution 1: Fix maxPossible Calculation

**Option A: Per-Rule Max**
```go
// Calculate max achievable for each rule type separately
for _, rule := range intent.Rules {
    maxPerRule := rule.Weight // Max for ONE keyword/phrase match
    r.maxPossible[intent.Name] += maxPerRule
}
```

**Option B: Expected Match Count**
```go
// Estimate realistic max (e.g., 3-4 keywords + 1-2 phrases)
maxKeywords := 3 * keywordWeight
maxPhrases := 1 * phraseWeight
r.maxPossible[intent.Name] = maxKeywords + maxPhrases
```

**Option C: Relative Score (Recommended)**
```go
// Don't use maxPossible at all
// Use absolute score threshold instead
if best.score < minScoreThreshold {
    return unknown
}
```

---

### Solution 2: Fix Phrase Matching

**Option A: Lemmatize Config Phrases**
```go
// During compile(), lemmatize phrase tokens
item.tokens = p.lemmatizer.lemmatize(strings.Fields(raw))
```

**Option B: Remove Stopwords from Config**
```go
// During compile(), remove stopwords from phrase tokens
item.tokens = removeStopwords(strings.Fields(raw))
```

**Option C: Store Both Forms**
```go
type compiledValue struct {
    raw            string
    tokens         []string  // Original (with stopwords)
    tokensClean    []string  // Without stopwords (for matching)
}
```

**Recommended:** Option B (remove stopwords from config phrases during compilation)

---

### Solution 3: Fix Ambiguity Detection

```go
// Don't multiply by maxPossible
// Use relative difference instead
delta := (best.score - second.score) / best.score
if delta < r.cfg.Threshold.AmbiguityDelta {
    // Ambiguous
}
```

Or use absolute difference:
```go
if best.score-second.score < r.cfg.Threshold.AmbiguityDelta {
    // Ambiguous
}
```

---

### Solution 4: Update rules.json

**Remove stopwords from phrases:**
```json
{
  "type": "phrase",
  "values": [
    "связаться оператор",      // Not "связаться с оператором"
    "поговорить человек",      // Not "поговорить с человеком"
    "добрый день"              // OK (no stopwords)
  ]
}
```

**Use lemmatized forms:**
```json
{
  "values": [
    "оператор",     // Not "оператор", "оператору", "оператором"
    "связаться",    // Lemmatized form
    "поговорить"    // Lemmatized form
  ]
}
```

---

## Priority Order

1. **HIGH:** Fix maxPossible calculation (Solution 1C - relative score)
2. **HIGH:** Fix phrase matching (Solution 2B - remove stopwords from config)
3. **MEDIUM:** Fix ambiguity detection (Solution 3)
4. **MEDIUM:** Update rules.json (Solution 4)
5. **LOW:** Add comprehensive tests for edge cases

---

## Testing Strategy

After fixes, verify:
1. ✅ Single keyword match: confidence > 0.5
2. ✅ Multiple keywords: confidence increases with each match
3. ✅ Phrase match without stopwords: works correctly
4. ✅ Ambiguous cases: detected and logged
5. ✅ Threshold min_confidence = 0.3: filters out weak matches
6. ✅ Threshold ambiguity_delta = 0.15: triggers appropriately