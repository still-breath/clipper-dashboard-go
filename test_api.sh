#!/bin/bash

# CCTV Backend API Test Script
# This script tests all the API endpoints

BASE_URL="http://localhost/api/v1"
# For direct backend testing: BASE_URL="http://localhost:5009/api/v1"

echo "🎯 Testing CCTV Backend API"
echo "=============================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
    fi
}

# Function to make API request and check status
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    
    echo -e "\n${YELLOW}Testing${NC}: $description"
    echo "Request: $method $endpoint"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint")
    fi
    
    # Split response and status code
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    echo "Response Code: $status_code"
    echo "Response Body: $body"
    
    if [ "$status_code" -eq "$expected_status" ]; then
        print_result 0 "$description"
        return 0
    else
        print_result 1 "$description (Expected: $expected_status, Got: $status_code)"
        return 1
    fi
}

# Function to test file upload
test_file_upload() {
    local booking_hour_id=$1
    local description="File Upload Test"
    
    echo -e "\n${YELLOW}Testing${NC}: $description"
    
    # Create a test video file (empty MP4)
    test_file="/tmp/test_video.mp4"
    echo "Creating test video file..."
    # Create a minimal MP4 file (this is just for testing - not a real video)
    printf "\x00\x00\x00\x20ftypmp4\x00\x00\x00\x00mp41\x00\x00\x00\x00\x00\x00\x00\x00mdat" > "$test_file"
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -F "video=@$test_file" \
        -F "bookingHourId=$booking_hour_id" \
        "$BASE_URL/clips/upload")
    
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    echo "Response Code: $status_code"
    echo "Response Body: $body"
    
    # Cleanup
    rm -f "$test_file"
    
    if [ "$status_code" -eq 201 ]; then
        print_result 0 "$description"
        return 0
    else
        print_result 1 "$description (Expected: 201, Got: $status_code)"
        return 1
    fi
}

# Start testing
echo -e "\n1️⃣ Testing Health Check"
test_endpoint "GET" "/health" "" 200 "Health Check"

echo -e "\n2️⃣ Testing Courts API"
test_endpoint "GET" "/courts" "" 200 "Get All Courts"

test_endpoint "POST" "/courts" '{"name": "Test Court", "description": "Test court for API testing"}' 201 "Create New Court"

test_endpoint "GET" "/courts?name=Test Court" "" 200 "Get Court by Name"

echo -e "\n3️⃣ Testing Booking Hours API"
test_endpoint "GET" "/booking-hours" "" 200 "Get All Booking Hours"

# Create a booking hour (we'll use court ID 1, assuming it exists from initial data)
current_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
end_time=$(date -u -d "+1 hour" +"%Y-%m-%dT%H:%M:%SZ")

booking_data="{\"courtId\": 1, \"dateStart\": \"$current_time\", \"dateEnd\": \"$end_time\", \"status\": \"active\"}"
test_endpoint "POST" "/booking-hours" "$booking_data" 201 "Create New Booking Hour"

# Get the created booking hour ID (this is a simplified approach)
booking_response=$(curl -s -X GET "$BASE_URL/booking-hours?courtId=1")
booking_hour_id=$(echo "$booking_response" | grep -o '"id":[0-9]*' | head -n1 | cut -d':' -f2)

if [ -n "$booking_hour_id" ]; then
    echo "Found booking hour ID: $booking_hour_id"
    test_endpoint "GET" "/booking-hours?courtId=1" "" 200 "Get Booking Hours by Court ID"
    
    echo -e "\n4️⃣ Testing Clips API"
    test_endpoint "GET" "/clips" "" 200 "Get All Clips"
    
    # Test file upload if we have a booking hour ID
    if command -v printf >/dev/null 2>&1; then
        test_file_upload "$booking_hour_id"
    else
        echo -e "${YELLOW}⚠️  Skipping file upload test (printf not available)${NC}"
    fi
    
    test_endpoint "GET" "/clips?bookingHourId=$booking_hour_id" "" 200 "Get Clips by Booking Hour ID"
else
    echo -e "${RED}❌ Could not find booking hour ID for file upload test${NC}"
fi

echo -e "\n5️⃣ Testing Error Handling"
test_endpoint "GET" "/courts?name=NonExistentCourt" "" 404 "Get Non-existent Court"

test_endpoint "POST" "/courts" '{"name": ""}' 400 "Create Court with Empty Name"

test_endpoint "POST" "/booking-hours" '{"courtId": 999, "dateStart": "'$current_time'", "dateEnd": "'$end_time'"}' 400 "Create Booking Hour with Invalid Court ID"

echo -e "\n📊 Test Summary"
echo "===================="
echo "✅ All tests completed!"
echo "Check the results above for any failures."
echo ""
echo "💡 Tips:"
echo "- Make sure the backend service is running"
echo "- Check if the database is properly initialized"
echo "- Verify network connectivity to the API endpoints"
echo ""
echo "🔧 If tests fail:"
echo "- Check service logs: make docker-logs"
echo "- Verify service health: curl $BASE_URL/health"
echo "- Check database connection: make docker-exec-db"