#!/bin/bash

BASE_URL="http://localhost:8081/api"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

pass() { echo -e "${GREEN}OK${NC} $1"; }
fail() { echo -e "${RED}FAIL${NC} $1"; }
info() { echo -e "${YELLOW}->${NC} $1"; }
section() { echo -e "${BLUE}==>${NC} $1"; }

section "COMPREHENSIVE CONTRIBUTOR TESTS"
echo ""

SIH_ID="01e331eb-0e30-4ada-adf9-fc5824f82675"
STUDENT_001_ID="300b39cd-422b-40da-aacf-87573457aa26"
STUDENT_004_ID="f20050f9-c23d-4f35-a7f8-e527d841b68f"

# Login as 23bsm001
section "Login as 23bsm001 (SIH member)"
TOKEN_001=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"23bsm001@iiitdmj.ac.in","password":"test123"}' | jq -r '.data.access_token')
[ -n "$TOKEN_001" ] && [ "$TOKEN_001" != "null" ] && pass "Logged in" || { fail "Login failed"; exit 1; }

# Get members
section "Get initial SIH members"
MEMBERS=$(curl -s -X GET "$BASE_URL/student/contributors?type=achievement&id=$SIH_ID" \
  -H "Authorization: Bearer $TOKEN_001")
COUNT=$(echo "$MEMBERS" | jq '[.data[]?] | length')
pass "SIH has $COUNT members"
echo "$MEMBERS" | jq '.data[] | {roll_no, name}'

# Test 1: Add 23bsm004
section "Test 1: Owner adds 23bsm004 to SIH"
ADD=$(curl -s -X POST "$BASE_URL/student/achievements/contributors" \
  -H "Authorization: Bearer $TOKEN_001" \
  -H "Content-Type: application/json" \
  -d "{\"achievement_id\":\"$SIH_ID\",\"student_id\":\"$STUDENT_004_ID\"}")
echo "$ADD" | jq -e '.success' > /dev/null && pass "Added 23bsm004" || fail "Failed: $(echo $ADD | jq -r '.error // .')"

# Verify
UPDATED=$(curl -s -X GET "$BASE_URL/student/contributors?type=achievement&id=$SIH_ID" \
  -H "Authorization: Bearer $TOKEN_001")
NEW_COUNT=$(echo "$UPDATED" | jq '[.data[]?] | length')
[ "$NEW_COUNT" -gt "$COUNT" ] && pass "Count increased: $COUNT -> $NEW_COUNT" || fail "Count did not increase"

# Test 2: 23bsm004 can see SIH
section "Test 2: 23bsm004 sees SIH in workspace"
TOKEN_004=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"23bsm004@iiitdmj.ac.in","password":"test123"}' | jq -r '.data.access_token')
WORK=$(curl -s -X GET "$BASE_URL/student/me" -H "Authorization: Bearer $TOKEN_004")
HAS=$(echo "$WORK" | jq "[.data.linked_records.achievement_members[]? | select(.achievement_id == \"$SIH_ID\")] | length")
[ "$HAS" -gt 0 ] && pass "23bsm004 sees SIH!" || fail "23bsm004 does NOT see SIH"

# Test 3: Remove 23bsm004
section "Test 3: Owner removes 23bsm004"
REM=$(curl -s -X DELETE "$BASE_URL/student/achievements/contributors" \
  -H "Authorization: Bearer $TOKEN_001" \
  -H "Content-Type: application/json" \
  -d "{\"achievement_id\":\"$SIH_ID\",\"student_id\":\"$STUDENT_004_ID\"}")
echo "$REM" | jq -e '.success' > /dev/null && pass "Removed 23bsm004" || fail "Failed: $(echo $REM | jq -r '.error // .')"

# Test 4: Non-owner cannot add
section "Test 4: Non-owner (23bsm004) cannot add (security)"
BAD_ADD=$(curl -s -X POST "$BASE_URL/student/achievements/contributors" \
  -H "Authorization: Bearer $TOKEN_004" \
  -H "Content-Type: application/json" \
  -d "{\"achievement_id\":\"$SIH_ID\",\"student_id\":\"$STUDENT_001_ID\"}")
echo "$BAD_ADD" | jq -e '.error' > /dev/null && pass "Non-owner blocked" || fail "SECURITY ISSUE!"

# Test 5: Search students
section "Test 5: Search students"
SEARCH=$(curl -s -X GET "$BASE_URL/student/search?q=23bsm&limit=3" \
  -H "Authorization: Bearer $TOKEN_001")
echo "$SEARCH" | jq -e '.success' > /dev/null && pass "Search works" || fail "Search failed"

echo ""
section "DONE"
