// Banking Ledger MongoDB initialization script

// Use the ledger database
db = db.getSiblingDB('ledger');

// Create the transactions collection with validation
db.createCollection('transactions', {
    validator: {
        $jsonSchema: {
            bsonType: 'object',
            required: ['_id', 'type', 'amount', 'currency', 'status', 'created_at'],
            properties: {
                _id: {
                    bsonType: 'string',
                    description: 'must be a string and is required'
                },
                type: {
                    enum: ['deposit', 'withdrawal', 'transfer'],
                    description: 'must be one of the enum values and is required'
                },
                from_account_id: {
                    bsonType: ['string', 'null'],
                    description: 'must be a string or null'
                },
                to_account_id: {
                    bsonType: ['string', 'null'],
                    description: 'must be a string or null'
                },
                amount: {
                    bsonType: 'double',
                    minimum: 0,
                    description: 'must be a positive number and is required'
                },
                currency: {
                    bsonType: 'string',
                    minLength: 3,
                    maxLength: 3,
                    description: 'must be a 3-character currency code and is required'
                },
                status: {
                    enum: ['pending', 'completed', 'failed', 'cancelled'],
                    description: 'must be one of the enum values and is required'
                },
                description: {
                    bsonType: 'string',
                    description: 'must be a string'
                },
                reference: {
                    bsonType: 'string',
                    description: 'must be a string'
                },
                metadata: {
                    bsonType: 'object',
                    description: 'must be an object'
                },
                created_at: {
                    bsonType: 'date',
                    description: 'must be a date and is required'
                },
                updated_at: {
                    bsonType: 'date',
                    description: 'must be a date'
                },
                processed_at: {
                    bsonType: ['date', 'null'],
                    description: 'must be a date or null'
                },
                error_message: {
                    bsonType: 'string',
                    description: 'must be a string'
                }
            }
        }
    }
});

// Create indexes for better query performance
db.transactions.createIndex({ 'from_account_id': 1 });
db.transactions.createIndex({ 'to_account_id': 1 });
db.transactions.createIndex({ 'type': 1 });
db.transactions.createIndex({ 'status': 1 });
db.transactions.createIndex({ 'created_at': -1 });
db.transactions.createIndex({ 'from_account_id': 1, 'created_at': -1 });
db.transactions.createIndex({ 'to_account_id': 1, 'created_at': -1 });
db.transactions.createIndex({ 'currency': 1 });
db.transactions.createIndex({ 'reference': 1 });

// Create compound indexes for common query patterns
db.transactions.createIndex({ 'status': 1, 'created_at': -1 });
db.transactions.createIndex({ 'type': 1, 'status': 1 });

// Insert some sample transactions for testing (optional)
db.transactions.insertMany([
    {
        '_id': '550e8400-e29b-41d4-a716-446655440101',
        'type': 'deposit',
        'to_account_id': '550e8400-e29b-41d4-a716-446655440001',
        'amount': 1000.00,
        'currency': 'USD',
        'status': 'completed',
        'description': 'Initial deposit',
        'reference': 'DEP001',
        'created_at': new Date(),
        'updated_at': new Date(),
        'processed_at': new Date()
    },
    {
        '_id': '550e8400-e29b-41d4-a716-446655440102',
        'type': 'deposit',
        'to_account_id': '550e8400-e29b-41d4-a716-446655440002',
        'amount': 2500.50,
        'currency': 'USD',
        'status': 'completed',
        'description': 'Initial deposit',
        'reference': 'DEP002',
        'created_at': new Date(),
        'updated_at': new Date(),
        'processed_at': new Date()
    }
]);

print('MongoDB initialization completed successfully!');