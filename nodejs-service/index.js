const express = require('express');
const sqlite3 = require('sqlite3').verbose();
const axios = require('axios');
const winston = require('winston');

// Winston ロガーの設定
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.json()
  ),
  transports: [
    new winston.transports.Console()
  ]
});

const app = express();
app.use(express.json());

// CORS
app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Headers', '*');
  res.header('Access-Control-Allow-Methods', '*');
  next();
});

const DB_PATH = '/data/inventory.db';

// Initialize database
const db = new sqlite3.Database(DB_PATH);
db.serialize(() => {
  db.run(`
    CREATE TABLE IF NOT EXISTS inventory (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      product_name TEXT NOT NULL UNIQUE,
      quantity INTEGER NOT NULL,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // Insert sample data
  db.run(`INSERT OR IGNORE INTO inventory (id, product_name, quantity) VALUES (1, 'Laptop', 50)`);
  db.run(`INSERT OR IGNORE INTO inventory (id, product_name, quantity) VALUES (2, 'Mouse', 200)`);
  db.run(`INSERT OR IGNORE INTO inventory (id, product_name, quantity) VALUES (3, 'Keyboard', 150)`);
});

app.get('/', (req, res) => {
  logger.info('Node.js service root endpoint called');

  res.json({ service: 'nodejs-express', status: 'running' });
});

app.post('/inventory/check', (req, res) => {
  const { product_name, quantity } = req.body;

  logger.info('Checking inventory', { product_name, quantity });

  db.get(
    'SELECT * FROM inventory WHERE product_name = ?',
    [product_name],
    (err, row) => {
      if (err) {
        logger.error('Database error', { error: err.message });
        return res.status(500).json({ error: err.message });
      }

      if (!row) {
        logger.warn('Product not found', { product_name });
        return res.json({ available: false, message: 'Product not found' });
      }

      const available = row.quantity >= quantity;
      logger.info('Inventory check result', { available, product_name });

      res.json({
        available,
        product_name: row.product_name,
        available_quantity: row.quantity,
        requested_quantity: quantity,
      });
    }
  );
});

app.get('/inventory', (req, res) => {
  logger.info('Fetching all inventory');

  db.all('SELECT * FROM inventory', [], (err, rows) => {
    if (err) {
      logger.error('Database error', { error: err.message });
      return res.status(500).json({ error: err.message });
    }

    logger.info('Retrieved inventory items', { count: rows.length });
    res.json({ inventory: rows });
  });
});

app.post('/inventory/reserve', async (req, res) => {
  const { product_name, quantity } = req.body;

  logger.info('Reserving inventory', { product_name, quantity });

  // Call Go service for pricing
  try {
    const response = await axios.post('http://go-service:8080/pricing/calculate', {
      product_name,
      quantity,
    });
    const pricingResult = response.data;
    logger.info('Pricing calculated', { pricingResult });

    res.json({
      reserved: true,
      pricing: pricingResult,
    });
  } catch (error) {
    logger.error('Failed to get pricing', { error: error.message });
    res.status(500).json({ error: error.message });
  }
});

app.get('/health', (req, res) => {
  res.json({ status: 'healthy' });
});

app.get('/error', (req, res) => {
  logger.error('Intentional error triggered');
  res.status(500).json({
    error: 'Intentional error for testing'
  });
});

app.post('/inventory/reserve/error', async (req, res) => {
  const { product_name, quantity } = req.body;

  logger.info('Reserving inventory with error test', { product_name, quantity });

  // Call Go service for pricing with error endpoint
  try {
    const response = await axios.post('http://go-service:8080/pricing/calculate/error', {
      product_name,
      quantity,
    });
    const pricingResult = response.data;
    logger.error('Pricing calculated with error', { pricingResult });

    res.status(500).json({
      reserved: false,
      pricing_error: pricingResult,
      error: 'Error occurred during pricing calculation'
    });
  } catch (error) {
    logger.error('Failed to get pricing (expected)', { error: error.message });
    res.status(500).json({
      error: error.message,
      message: 'Error in pricing service call'
    });
  }
});

const PORT = 3000;
app.listen(PORT, () => {
  logger.info('Node.js service started', { port: PORT });
});

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM received, shutting down gracefully');
  db.close();
  process.exit(0);
});