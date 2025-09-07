#!/bin/bash

# PostgreSQL Data Visualization Script for Banking Ledger

echo "=== Banking Ledger PostgreSQL Data Viewer ==="
echo

# Database connection
DB_NAME="banking_ledger"
PSQL_CMD="/opt/homebrew/opt/postgresql@15/bin/psql"

# Check if database is accessible
if ! $PSQL_CMD -d $DB_NAME -c "SELECT 1;" > /dev/null 2>&1; then
    echo "Error: Cannot connect to PostgreSQL database '$DB_NAME'"
    echo "Make sure PostgreSQL is running: brew services start postgresql@15"
    exit 1
fi

echo "✅ Connected to PostgreSQL database: $DB_NAME"
echo

# Function to run query and display results
run_query() {
    local title="$1"
    local query="$2"
    
    echo "📊 $title"
    echo "$(printf '=%.0s' {1..60})"
    $PSQL_CMD -d $DB_NAME -c "$query"
    echo
}

# Display all accounts
run_query "ALL ACCOUNTS" "
SELECT 
    id,
    user_id,
    balance,
    currency,
    status,
    created_at,
    version
FROM accounts 
ORDER BY created_at DESC;
"

# Display account summary by currency
run_query "ACCOUNT SUMMARY BY CURRENCY" "
SELECT 
    currency,
    COUNT(*) as account_count,
    SUM(balance) as total_balance,
    AVG(balance) as avg_balance,
    MIN(balance) as min_balance,
    MAX(balance) as max_balance
FROM accounts 
GROUP BY currency
ORDER BY currency;
"

# Display account summary by user
run_query "ACCOUNT SUMMARY BY USER" "
SELECT 
    user_id,
    COUNT(*) as account_count,
    STRING_AGG(currency, ', ') as currencies,
    SUM(balance) as total_balance_all_currencies
FROM accounts 
GROUP BY user_id
ORDER BY user_id;
"

# Display recent accounts
run_query "RECENTLY CREATED ACCOUNTS (Last 10)" "
SELECT 
    user_id,
    balance,
    currency,
    status,
    created_at
FROM accounts 
ORDER BY created_at DESC 
LIMIT 10;
"

# Display account statistics
run_query "ACCOUNT STATISTICS" "
SELECT 
    COUNT(*) as total_accounts,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT currency) as currencies_supported,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_accounts,
    SUM(CASE WHEN status = 'inactive' THEN 1 ELSE 0 END) as inactive_accounts
FROM accounts;
"

echo "💡 Tips:"
echo "- To connect manually: $PSQL_CMD -d $DB_NAME"
echo "- To export data: $PSQL_CMD -d $DB_NAME -c \"COPY accounts TO '/tmp/accounts.csv' WITH CSV HEADER;\""
echo "- To view table structure: $PSQL_CMD -d $DB_NAME -c \"\\d accounts\""
echo