#!/bin/bash

# MongoDB Data Visualization Script for Banking Ledger

echo "=== Banking Ledger MongoDB Data Viewer ==="
echo

# Database connection
DB_NAME="ledger"
COLLECTION="transactions"

# Check if MongoDB is accessible
if ! mongosh --eval "db.runCommand('ping')" --quiet > /dev/null 2>&1; then
    echo "Error: Cannot connect to MongoDB"
    echo "Make sure MongoDB is running: brew services start mongodb/brew/mongodb-community"
    exit 1
fi

echo "✅ Connected to MongoDB database: $DB_NAME"
echo

# Function to run MongoDB query
run_mongo_query() {
    local title="$1"
    local query="$2"
    
    echo "📊 $title"
    echo "$(printf '=%.0s' {1..60})"
    mongosh $DB_NAME --eval "$query" --quiet
    echo
}

# Display all transactions
run_mongo_query "ALL TRANSACTIONS" "
db.$COLLECTION.find({}).sort({created_at: -1}).limit(20).forEach(
    function(doc) {
        print(JSON.stringify({
            id: doc._id,
            type: doc.type,
            from_account: doc.from_account_id || 'N/A',
            to_account: doc.to_account_id || 'N/A',
            amount: doc.amount,
            currency: doc.currency,
            status: doc.status,
            created_at: doc.created_at
        }, null, 2));
        print('---');
    }
);
"

# Display transaction summary by type
run_mongo_query "TRANSACTION SUMMARY BY TYPE" "
db.$COLLECTION.aggregate([
    {
        \$group: {
            _id: '\$type',
            count: { \$sum: 1 },
            total_amount: { \$sum: '\$amount' },
            avg_amount: { \$avg: '\$amount' },
            min_amount: { \$min: '\$amount' },
            max_amount: { \$max: '\$amount' }
        }
    },
    {
        \$sort: { count: -1 }
    }
]).forEach(function(doc) {
    print(JSON.stringify(doc, null, 2));
});
"

# Display transaction summary by status
run_mongo_query "TRANSACTION SUMMARY BY STATUS" "
db.$COLLECTION.aggregate([
    {
        \$group: {
            _id: '\$status',
            count: { \$sum: 1 },
            total_amount: { \$sum: '\$amount' }
        }
    },
    {
        \$sort: { count: -1 }
    }
]).forEach(function(doc) {
    print(JSON.stringify(doc, null, 2));
});
"

# Display transaction summary by currency
run_mongo_query "TRANSACTION SUMMARY BY CURRENCY" "
db.$COLLECTION.aggregate([
    {
        \$group: {
            _id: '\$currency',
            count: { \$sum: 1 },
            total_amount: { \$sum: '\$amount' },
            avg_amount: { \$avg: '\$amount' }
        }
    },
    {
        \$sort: { total_amount: -1 }
    }
]).forEach(function(doc) {
    print(JSON.stringify(doc, null, 2));
});
"

# Display recent transactions
run_mongo_query "RECENT TRANSACTIONS (Last 10)" "
db.$COLLECTION.find({}).sort({created_at: -1}).limit(10).forEach(
    function(doc) {
        print(
            doc.created_at + ' | ' +
            doc.type.toUpperCase() + ' | ' +
            doc.amount + ' ' + doc.currency + ' | ' +
            doc.status.toUpperCase() + ' | ' +
            (doc.from_account_id || 'N/A') + ' -> ' +
            (doc.to_account_id || 'N/A')
        );
    }
);
"

# Display transaction statistics
run_mongo_query "TRANSACTION STATISTICS" "
var stats = db.$COLLECTION.aggregate([
    {
        \$group: {
            _id: null,
            total_transactions: { \$sum: 1 },
            total_amount: { \$sum: '\$amount' },
            avg_amount: { \$avg: '\$amount' },
            unique_accounts: { \$addToSet: { \$cond: [
                { \$ne: ['\$from_account_id', null] }, '\$from_account_id', '\$to_account_id'
            ]}},
            pending_count: { \$sum: { \$cond: [{ \$eq: ['\$status', 'pending'] }, 1, 0] }},
            completed_count: { \$sum: { \$cond: [{ \$eq: ['\$status', 'completed'] }, 1, 0] }},
            failed_count: { \$sum: { \$cond: [{ \$eq: ['\$status', 'failed'] }, 1, 0] }}
        }
    }
]).toArray()[0];

if (stats) {
    print('Total Transactions: ' + stats.total_transactions);
    print('Total Amount: ' + stats.total_amount.toFixed(2));
    print('Average Amount: ' + stats.avg_amount.toFixed(2));
    print('Unique Accounts: ' + stats.unique_accounts.length);
    print('Pending: ' + stats.pending_count);
    print('Completed: ' + stats.completed_count);
    print('Failed: ' + stats.failed_count);
} else {
    print('No transactions found');
}
"

echo "💡 Tips:"
echo "- To connect manually: mongosh $DB_NAME"
echo "- To export data: mongoexport --db=$DB_NAME --collection=$COLLECTION --out=/tmp/transactions.json"
echo "- To view collection info: mongosh $DB_NAME --eval \"db.$COLLECTION.stats()\""
echo