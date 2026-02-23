const { MongoClient } = require('mongodb');


async function run() {
  try {
    const mongoURI = process.env.MONGODB_URI;
    const dbName = process.env.MONGODB_DB_NAME;
    const collectionName = process.env.MONGODB_COLLECTION_NAME;
    const metricsDataJson = process.env.METRICS_DATA;

    if (!mongoURI || !dbName || !collectionName) {
      console.error('Missing one or more required environment variables: MONGODB_URI, MONGODB_DB_NAME, MONGODB_COLLECTION_NAME');
      process.exit(1);
    }

    let metricsData;
    try {
      metricsData = JSON.parse(metricsDataJson);
    } catch (e) {
      console.error(`Failed to parse metrics data JSON: ${e.message}`);
      process.exit(1);
    }

    console.log('Connecting to MongoDB...');
    const client = new MongoClient(mongoURI);
    await client.connect();
    console.log('Connected to MongoDB.');

    const database = client.db(dbName);
    const collection = database.collection(collectionName);

    console.log('Inserting metrics data...');
    const result = await collection.insertOne(metricsData);
    console.log(`Successfully inserted document with _id: ${result.insertedId}`);

  } catch (error) {
    console.error(`Failed to send metrics to MongoDB: ${error.message}`);
    process.exit(1);
  } finally {
    await client.close();
    console.log('MongoDB connection closed.');
  }
}

// Export the run function for external use
module.exports = {
  run
};


