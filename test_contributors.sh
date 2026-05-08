#!/bin/bash

BASE_URL="http://localhost:8081/api"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; }
info() { echo -e "${YELLOW}→${NC} $1"; }

SIH_ID="01e331eb-0e30-4ada-adf9-fc5824f82675"
STUDENT_001_ID="300b39cd-422b-40da-aacf-87573457aa26"  # 23bsm001
STUDENT_002_ID="58e683ee-f25a-42ad-89db-891af24bfcdb"  # 23bsm002
STUDENT_003_ID="1ed87d8a-b2d5-4128-989d-6a60b86a02f6"  # 23bsm003
TEST_USER_ID="1eade87b-9bc7-463e-b7aa-24cf5d122c30"    # test@iiitdmj.ac.in
TEST_STUDENT_ID="3fd7d211-1a66-41af-92d2-7e9a721d4442" # 23bsm051

info "SIH Achievement ID: $SIH_ID"
info "Test user: test@iiitdmj.ac.in (student_id=$TEST_STUDENT_ID)"

# Login as test user
info "Logging in as test@iiitdmj.ac.in..."
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@iiitdmj.ac.in","password":"test123"}')

# Try access_token first, then token
TEST_TOKEN=$(echo $LOGIN_RESP | jq -r '.data.access_token // .data.token // empty')

if [ -z "$TEST_TOKEN" ] || [ "$TEST_TOKEN" = "null" ]; then
  fail "Failed to get token. Response: $LOGIN_RESP"
  exit 1
fi

pass "Login successful"

# Get current members of SIH
info "Getting current members of SIH achievement..."
CONTRIB_RESP=$(curl -s -X GET "$BASE_URL/student/contributors?type=achievement&id=$SIH_ID" \
  -H "Authorization: Bearer $TEST_TOKEN")
echo "Contributors response: $CONTRIB_RESP"

if echo "$CONTRIB_RESP" | jq -e '.success' > /dev/null; then
  pass "Successfully retrieved contributors"
  echo "$CONTRIB_RESP" | jq '.data'
else
  fail "Failed to get contributors: $CONTRIB_RESP"
fi

# Check if test user is a member of SIH
info "Checking if test user (23bsm051) is a member of SIH..."
IS_MEMBER=$(echo "$CONTRIB_RESP" | jq "[.data[]? | select(.student_id == \"$TEST_STUDENT_ID\")] | length")
if [ "$IS_MEMBER" -gt 0 ]; then
  pass "Test user IS a member of SIH"
  
  # Test: Try to add a new contributor
  info "Test: Adding 23bsm003 to SIH (as test user who is a member)..."
  ADD_RESP=$(curl -s -X POST "$BASE_URL/student/achievements/contributors" \
    -H "Authorization: Bearer $TEST_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"achievement_id\":\"$SIH_ID\",\"student_id\":\"$STUDENT_003_ID\"}")
  echo "Add contributor response: $ADD_RESP"
  
  if echo "$ADD_RESP" | jq -e '.success' > /dev/null; then
    pass "Successfully added 23bsm003 as contributor"
  elif echo "$ADD_RESP" | jq -e '.error' > /dev/null; then
    ERR_MSG=$(echo "$ADD_RESP" | jq -r '.error')
    if echo "$ERR_MSG" | grep -qi "not an owner"; then
      fail "Not authorized - need to be an owner (not just a member)"
    else
      fail "Failed to add: $ERR_MSG"
    fi
  else
    fail "Unexpected response: $ADD_RESP"
  fi
else
  info "Test user is NOT a member of SIH, checking workspace..."
  
  # Get workspace to see what achievements test user has
  WORKSPACE_RESP=$(curl -s -X GET "$BASE_URL/student/me" \
    -H "Authorization: Bearer $TEST_TOKEN")
  
  if echo "$WORKSPACE_RESP" | jq -e '.success' > /dev/null; then
    ACH_COUNT=$(echo "$WORKSPACE_RESP" | jq '.data.achievements | length')
    pass "Workspace loaded (has $ACH_COUNT achievements)"
    echo "Achievements in workspace:"
    echo "$WORKSPACE_RESP" | jq '.data.achievements[] | {id, title}'
  else
    fail "Failed to get workspace: $WORKSPACE_RESP"
  fi
fi

# Test 23bsm001's workspace
info "Test: Check 23bsm001's workspace (they should have SIH)..."
LOGIN_001_RESP=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"23bsm001@iiitdmj.ac.in","password":"student123"}')
TOKEN_001=$(echo $LOGIN_001_RESP | jq -r '.data.access_token // .data.token // empty')

if [ -n "$TOKEN_001" ] && [ "$TOKEN_001" != "null" ]; then
  pass "23bsm001 login successful"
  
  WORKSPACE_001_RESP=$(curl -s -X GET "$BASE_URL/student/me" \
    -H "Authorization: Bearer $TOKEN_001")
  
  if echo "$WORKSPACE_001_RESP" | jq -e '.success' > /dev/null; then
    ACH_COUNT=$(echo "$WORKSPACE_001_RESP" | jq '.data.achievements | length')
    pass "23bsm001 workspace loaded (has $ACH_COUNT achievements)"
    
    # Check for SIH in workspace
    HAS_SIH=$(echo "$WORKSPACE_001_RESP" | jq "[.data.achievements[] | select(.id == \"$SIH_ID\")] | length")
    if [ "$HAS_SIH" -gt 0 ]; then
      pass "SIH achievement IS in 23bsm001's workspace!"
      echo "$WORKSPACE_001_RESP" | jq ".data.achievements[] | select(.id == \"$SIH_ID\")"
    else
      fail "SIH achievement NOT in 23bsm001's workspace - THIS IS THE BUG!"
      echo "Achievements in 23bsm001's workspace:"
      echo "$WORKSPACE_001_RESP" | jq '.data.achievements[] | {id, title}'
    fi
  else
    fail "Failed to get workspace: $WORKSPACE_001_RESP"
  fi
else
  fail "Failed to login as 23bsm001. Response: $LOGIN_001_RESP"
fi

# Try to add 23bsm003 as contributor (using test user who is a member)
info "Test: Adding 23bsm003 to SIH..."
ADD_RESP=$(curl -s -X POST "$BASE_URL/student/achievements/contributors" \
  -H "Authorization: Bearer $TEST_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"achievement_id\":\"$SIH_ID\",\"student_id\":\"$STUDENT_003_ID\"}")
echo "Response: $ADD_RESP"

if echo "$ADD_RESP" | jq -e '.success' > /dev/null; then
  pass "Successfully added 23bsm003 as contributor"
  
  # Verify the new member
  info "Verifying new member..."
  UPDATED_CONTRIB=$(curl -s -X GET "$BASE_URL/student/contributors?type=achievement&id=$SIH_ID" \
    -H "Authorization: Bearer $TEST_TOKEN")
  echo "$UPDATED_CONTRIB" | jq '.data'
elif echo "$ADD_RESP" | jq -e '.error' > /dev/null; then
  ERR_MSG=$(echo "$ADD_RESP" | jq -r '.error')
  fail "Failed to add contributor: $ERR_MSG"
fi

echo ""
echo "===================="
echo "Test Summary Complete"
echo "===================="
