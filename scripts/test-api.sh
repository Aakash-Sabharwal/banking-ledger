#!/bin/bash

# Banking Ledger Service Test Script

set -e

echo "=== Banking Ledger Service Test Script ==="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="http://localhost:8080/api/v1"
HEALTH_URL="http://localhost:8080/health"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if service is running
check_service() {
    print_status "Checking if service is running..."
    
    if curl -s "$HEALTH_URL" > /dev/null; then
        print_success "Service is running"
        return 0
    else
        print_error "Service is not responding"
        return 1
    fi
}

# Function to test account creation
test_account_creation() {
    print_status "Testing account creation..."
    
    response=$(curl -s -X POST "$API_URL/accounts" \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "testuser1",
            "initial_balance": 1000.00,
            "currency": "USD"
        }')
    
    if echo "$response" | grep -q '"id"'; then
        account_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        print_success "Account created successfully: $account_id"
        echo "$account_id" > /tmp/test_account_id
        return 0
    else
        print_error "Failed to create account: $response"
        return 1
    fi
}

# Function to test account retrieval
test_account_retrieval() {
    print_status "Testing account retrieval..."
    
    if [ ! -f /tmp/test_account_id ]; then
        print_error "No account ID found. Run account creation test first."
        return 1
    fi
    
    account_id=$(cat /tmp/test_account_id)
    response=$(curl -s "$API_URL/accounts/$account_id")
    
    if echo "$response" | grep -q '"balance"'; then
        balance=$(echo "$response" | grep -o '"balance":[0-9.]*' | cut -d':' -f2)
        print_success "Account retrieved successfully. Balance: $balance"
        return 0
    else
        print_error "Failed to retrieve account: $response"
        return 1
    fi
}

# Function to test deposit transaction
test_deposit_transaction() {
    print_status "Testing deposit transaction..."
    
    if [ ! -f /tmp/test_account_id ]; then
        print_error "No account ID found. Run account creation test first."
        return 1
    fi
    
    account_id=$(cat /tmp/test_account_id)
    response=$(curl -s -X POST "$API_URL/transactions" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"deposit\",
            \"to_account_id\": \"$account_id\",
            \"amount\": 500.00,
            \"currency\": \"USD\",
            \"description\": \"Test deposit\",
            \"reference\": \"TEST001\"
        }")
    
    if echo "$response" | grep -q '"id"'; then
        tx_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        print_success "Deposit transaction created: $tx_id"
        echo "$tx_id" > /tmp/test_transaction_id
        return 0
    else
        print_error "Failed to create deposit transaction: $response"
        return 1
    fi
}

# Function to test withdrawal transaction
test_withdrawal_transaction() {
    print_status "Testing withdrawal transaction..."
    
    if [ ! -f /tmp/test_account_id ]; then
        print_error "No account ID found. Run account creation test first."
        return 1
    fi
    
    account_id=$(cat /tmp/test_account_id)
    response=$(curl -s -X POST "$API_URL/transactions" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"withdrawal\",
            \"from_account_id\": \"$account_id\",
            \"amount\": 200.00,
            \"currency\": \"USD\",
            \"description\": \"Test withdrawal\",
            \"reference\": \"TEST002\"
        }")
    
    if echo "$response" | grep -q '"id"'; then
        tx_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        print_success "Withdrawal transaction created: $tx_id"
        return 0
    else
        print_error "Failed to create withdrawal transaction: $response"
        return 1
    fi
}

# Function to test transfer transaction
test_transfer_transaction() {
    print_status "Testing transfer transaction..."
    
    # Create second account first
    print_status "Creating second account for transfer..."
    response=$(curl -s -X POST "$API_URL/accounts" \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "testuser2",
            "initial_balance": 500.00,
            "currency": "USD"
        }')
    
    if ! echo "$response" | grep -q '"id"'; then
        print_error "Failed to create second account: $response"
        return 1
    fi
    
    account2_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    account1_id=$(cat /tmp/test_account_id)
    
    response=$(curl -s -X POST "$API_URL/transactions" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"transfer\",
            \"from_account_id\": \"$account1_id\",
            \"to_account_id\": \"$account2_id\",
            \"amount\": 150.00,
            \"currency\": \"USD\",
            \"description\": \"Test transfer\",
            \"reference\": \"TEST003\"
        }")
    
    if echo "$response" | grep -q '"id"'; then
        tx_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        print_success "Transfer transaction created: $tx_id"
        return 0
    else
        print_error "Failed to create transfer transaction: $response"
        return 1
    fi
}

# Function to test transaction history
test_transaction_history() {
    print_status "Testing transaction history retrieval..."
    
    if [ ! -f /tmp/test_account_id ]; then
        print_error "No account ID found. Run account creation test first."
        return 1
    fi
    
    account_id=$(cat /tmp/test_account_id)
    response=$(curl -s "$API_URL/accounts/$account_id/transactions?limit=10")
    
    if echo "$response" | grep -q '"transactions"'; then
        count=$(echo "$response" | grep -o '"count":[0-9]*' | cut -d':' -f2)
        print_success "Transaction history retrieved. Transaction count: $count"
        return 0
    else
        print_error "Failed to retrieve transaction history: $response"
        return 1
    fi
}

# Function to test error handling
test_error_handling() {
    print_status "Testing error handling..."
    
    # Test invalid amount
    response=$(curl -s -X POST "$API_URL/transactions" \
        -H "Content-Type: application/json" \
        -d '{
            "type": "deposit",
            "to_account_id": "invalid-id",
            "amount": -100.00,
            "currency": "USD"
        }')
    
    if echo "$response" | grep -q '"error"'; then
        print_success "Error handling works correctly"
        return 0
    else
        print_warning "Error handling might not be working as expected"
        return 1
    fi
}

# Function to clean up test data
cleanup() {
    print_status "Cleaning up test data..."
    rm -f /tmp/test_account_id /tmp/test_transaction_id
    print_success "Cleanup completed"
}

# Main test execution
main() {
    echo "Starting Banking Ledger Service tests..."
    echo "========================================="
    
    # Check if service is running
    if ! check_service; then
        print_error "Service is not running. Please start the service first."
        exit 1
    fi
    
    # Run tests
    tests=(
        "test_account_creation"
        "test_account_retrieval"
        "test_deposit_transaction"
        "test_withdrawal_transaction"
        "test_transfer_transaction"
        "test_transaction_history"
        "test_error_handling"
    )
    
    passed=0
    total=${#tests[@]}
    
    for test in "${tests[@]}"; do
        echo
        if $test; then
            ((passed++))
        fi
        sleep 1  # Brief pause between tests
    done
    
    echo
    echo "========================================="
    echo "Test Results:"
    echo "Passed: $passed/$total"
    
    if [ $passed -eq $total ]; then
        print_success "All tests passed! 🎉"
        cleanup
        exit 0
    else
        print_error "Some tests failed. Please check the logs."
        cleanup
        exit 1
    fi
}

# Handle script interruption
trap cleanup EXIT INT TERM

# Run main function
main "$@"