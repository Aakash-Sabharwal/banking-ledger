#!/bin/bash

# Combined Data Visualization Script for Banking Ledger Service

echo "🏦 Banking Ledger Service - Complete Data Viewer"
echo "=================================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_section() {
    echo -e "${BLUE}$1${NC}"
    echo "$(printf '=%.0s' {1..60})"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Check services
check_services() {
    print_section "SERVICE STATUS CHECK"
    
    # Check PostgreSQL
    if /opt/homebrew/opt/postgresql@15/bin/pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
        print_success "PostgreSQL is running"
    else
        print_error "PostgreSQL is not running"
        echo "Start with: brew services start postgresql@15"
        return 1
    fi
    
    # Check MongoDB
    if mongosh --eval "db.runCommand('ping')" --quiet > /dev/null 2>&1; then
        print_success "MongoDB is running"
    else
        print_error "MongoDB is not running"
        echo "Start with: brew services start mongodb/brew/mongodb-community"
        return 1
    fi
    
    # Check RabbitMQ
    if curl -s http://localhost:15672 > /dev/null 2>&1; then
        print_success "RabbitMQ is running"
    else
        print_warning "RabbitMQ may not be running (http://localhost:15672 not accessible)"
    fi
    
    echo
}

# Display menu
show_menu() {
    print_section "DATA VISUALIZATION OPTIONS"
    echo "1. View PostgreSQL Data (Accounts)"
    echo "2. View MongoDB Data (Transactions)"
    echo "3. View Combined Summary"
    echo "4. Export All Data"
    echo "5. Interactive PostgreSQL Shell"
    echo "6. Interactive MongoDB Shell"
    echo "7. RabbitMQ Management UI"
    echo "8. Generate Data Report"
    echo "9. Exit"
    echo
}

# PostgreSQL viewer
view_postgres_data() {
    print_section "POSTGRESQL DATA - ACCOUNTS"
    ./scripts/view-postgres-data.sh
}

# MongoDB viewer
view_mongo_data() {
    print_section "MONGODB DATA - TRANSACTIONS"
    ./scripts/view-mongo-data.sh
}

# Combined summary
view_combined_summary() {
    print_section "COMBINED DATA SUMMARY"
    
    echo "📊 ACCOUNT SUMMARY"
    /opt/homebrew/opt/postgresql@15/bin/psql banking_ledger -c "
    SELECT 
        COUNT(*) as total_accounts,
        COUNT(DISTINCT user_id) as unique_users,
        SUM(balance) as total_balance,
        currency
    FROM accounts 
    GROUP BY currency;"
    
    echo
    echo "📊 TRANSACTION SUMMARY"
    mongosh ledger --eval "
    db.transactions.aggregate([
        {
            \$group: {
                _id: null,
                total_transactions: { \$sum: 1 },
                total_amount: { \$sum: '\$amount' },
                pending: { \$sum: { \$cond: [{ \$eq: ['\$status', 'pending'] }, 1, 0] }},
                completed: { \$sum: { \$cond: [{ \$eq: ['\$status', 'completed'] }, 1, 0] }},
                failed: { \$sum: { \$cond: [{ \$eq: ['\$status', 'failed'] }, 1, 0] }}
            }
        }
    ]).forEach(function(doc) {
        print('Total Transactions: ' + doc.total_transactions);
        print('Total Amount: ' + doc.total_amount);
        print('Pending: ' + doc.pending);
        print('Completed: ' + doc.completed);
        print('Failed: ' + doc.failed);
    });
    " --quiet
    echo
}

# Export data
export_data() {
    print_section "EXPORTING DATA"
    
    mkdir -p ./data-exports
    
    # Export PostgreSQL data
    echo "Exporting PostgreSQL accounts..."
    /opt/homebrew/opt/postgresql@15/bin/psql banking_ledger -c "COPY accounts TO '$(pwd)/data-exports/accounts_$(date +%Y%m%d_%H%M%S).csv' WITH CSV HEADER;"
    
    # Export MongoDB data
    echo "Exporting MongoDB transactions..."
    mongoexport --db=ledger --collection=transactions --out=./data-exports/transactions_$(date +%Y%m%d_%H%M%S).json
    
    print_success "Data exported to ./data-exports/"
    ls -la ./data-exports/
    echo
}

# Interactive shells
postgres_shell() {
    print_section "POSTGRESQL INTERACTIVE SHELL"
    echo "💡 Useful commands:"
    echo "\\d accounts          - Describe accounts table"
    echo "\\l                   - List databases"
    echo "\\q                   - Quit"
    echo "SELECT * FROM accounts LIMIT 5; - View sample data"
    echo
    /opt/homebrew/opt/postgresql@15/bin/psql banking_ledger
}

mongo_shell() {
    print_section "MONGODB INTERACTIVE SHELL"
    echo "💡 Useful commands:"
    echo "db.transactions.find().limit(5)  - View sample data"
    echo "db.transactions.count()          - Count documents"
    echo "show collections                 - List collections"
    echo "exit                             - Quit"
    echo
    mongosh ledger
}

# RabbitMQ UI
rabbitmq_ui() {
    print_section "RABBITMQ MANAGEMENT UI"
    echo "Opening RabbitMQ Management UI..."
    echo "URL: http://localhost:15672"
    echo "Username: guest"
    echo "Password: guest"
    echo
    open http://localhost:15672 2>/dev/null || echo "Please open http://localhost:15672 in your browser"
}

# Generate report
generate_report() {
    print_section "GENERATING DATA REPORT"
    
    REPORT_FILE="./banking_ledger_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "Banking Ledger Service Data Report"
        echo "Generated: $(date)"
        echo "=================================="
        echo
        
        echo "=== ACCOUNT SUMMARY ==="
        /opt/homebrew/opt/postgresql@15/bin/psql banking_ledger -c "
        SELECT 
            currency,
            COUNT(*) as accounts,
            SUM(balance) as total_balance,
            AVG(balance) as avg_balance
        FROM accounts 
        GROUP BY currency;"
        
        echo
        echo "=== TRANSACTION SUMMARY ==="
        mongosh ledger --eval "
        db.transactions.aggregate([
            {
                \$group: {
                    _id: '\$type',
                    count: { \$sum: 1 },
                    total_amount: { \$sum: '\$amount' }
                }
            }
        ]).forEach(function(doc) {
            print(doc._id + ': ' + doc.count + ' transactions, Total: ' + doc.total_amount);
        });
        " --quiet
        
        echo
        echo "=== SYSTEM STATUS ==="
        echo "PostgreSQL: $(brew services list | grep postgresql@15 | awk '{print $2}')"
        echo "MongoDB: $(brew services list | grep mongodb-community | awk '{print $2}')"
        echo "RabbitMQ: $(brew services list | grep rabbitmq | awk '{print $2}')"
        
    } > "$REPORT_FILE"
    
    print_success "Report generated: $REPORT_FILE"
    echo "Preview:"
    head -20 "$REPORT_FILE"
    echo
}

# Main execution
main() {
    # Check services first
    if ! check_services; then
        echo "Please start the required services and try again."
        exit 1
    fi
    
    while true; do
        show_menu
        read -p "Select an option (1-9): " choice
        echo
        
        case $choice in
            1) view_postgres_data ;;
            2) view_mongo_data ;;
            3) view_combined_summary ;;
            4) export_data ;;
            5) postgres_shell ;;
            6) mongo_shell ;;
            7) rabbitmq_ui ;;
            8) generate_report ;;
            9) echo "Goodbye!"; exit 0 ;;
            *) print_error "Invalid option. Please select 1-9." ;;
        esac
        
        echo
        read -p "Press Enter to continue..."
        echo
    done
}

# Run main function
main "$@"