# Usage Examples

## Quick Start

### 1. Using cURL

```bash
# Basic usage (uses default config)
curl -X POST http://localhost:8080/api/process-invoice \
  -F "file=@receipt.jpg" \
  | jq .

# With specific AI provider
curl -X POST http://localhost:8080/api/process-invoice \
  -F "file=@receipt.jpg" \
  -F "aiProvider=gemini" \
  | jq .

# Using vision model (skip OCR)
curl -X POST http://localhost:8080/api/process-invoice \
  -F "file=@receipt.jpg" \
  -F "useVisionModel=true" \
  -F "aiProvider=openai" \
  -F "model=gpt-4-vision-preview" \
  | jq .

# Spanish language OCR
curl -X POST http://localhost:8080/api/process-invoice \
  -F "file=@factura.jpg" \
  -F "language=spa" \
  | jq .
```

### 2. Python

```python
import requests
import json

def process_invoice(image_path, use_vision=False):
    url = "http://localhost:8080/api/process-invoice"

    with open(image_path, "rb") as f:
        files = {"file": f}
        data = {
            "useVisionModel": str(use_vision).lower(),
            "aiProvider": "openai"
        }

        response = requests.post(url, files=files, data=data)
        return response.json()

# Example usage
result = process_invoice("invoice.jpg")

if result["success"]:
    invoice = result["invoice"]
    print(f"Vendor: {invoice['vendor']}")
    print(f"Total: ${invoice['total']}")
    print(f"Date: {invoice['date']}")
    print(f"\nItems:")
    for item in invoice.get("items", []):
        print(f"  - {item['name']}: ${item['amount']}")
    print(f"\nProcessing time: {result['totalDuration']:.2f}s")
else:
    print(f"Error: {result['error']}")
```

### 3. JavaScript/Node.js

```javascript
const FormData = require('form-data');
const fs = require('fs');
const axios = require('axios');

async function processInvoice(imagePath, options = {}) {
  const form = new FormData();
  form.append('file', fs.createReadStream(imagePath));

  if (options.aiProvider) {
    form.append('aiProvider', options.aiProvider);
  }
  if (options.useVisionModel) {
    form.append('useVisionModel', 'true');
  }
  if (options.model) {
    form.append('model', options.model);
  }

  const response = await axios.post(
    'http://localhost:8080/api/process-invoice',
    form,
    {
      headers: form.getHeaders()
    }
  );

  return response.data;
}

// Example usage
(async () => {
  try {
    const result = await processInvoice('invoice.jpg', {
      aiProvider: 'openai',
      useVisionModel: false
    });

    if (result.success) {
      console.log('Vendor:', result.invoice.vendor);
      console.log('Total:', result.invoice.total);
      console.log('Items:', result.invoice.items);
    } else {
      console.error('Error:', result.error);
    }
  } catch (error) {
    console.error('Request failed:', error.message);
  }
})();
```

### 4. Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type InvoiceResponse struct {
	Success       bool    `json:"success"`
	Invoice       Invoice `json:"invoice,omitempty"`
	Error         string  `json:"error,omitempty"`
	TotalDuration float64 `json:"totalDuration"`
}

type Invoice struct {
	Vendor     string  `json:"vendor"`
	Date       string  `json:"date"`
	Total      float64 `json:"total"`
	Tax        float64 `json:"tax"`
	Categories []string `json:"categories"`
}

func processInvoice(imagePath string) (*InvoiceResponse, error) {
	// Open file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", imagePath)
	if err != nil {
		return nil, err
	}
	io.Copy(part, file)

	// Add other fields
	writer.WriteField("aiProvider", "openai")
	writer.WriteField("useVisionModel", "false")

	writer.Close()

	// Make request
	req, err := http.NewRequest("POST", "http://localhost:8080/api/process-invoice", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var result InvoiceResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func main() {
	result, err := processInvoice("invoice.jpg")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if result.Success {
		fmt.Printf("Vendor: %s\n", result.Invoice.Vendor)
		fmt.Printf("Total: $%.2f\n", result.Invoice.Total)
		fmt.Printf("Processing time: %.2fs\n", result.TotalDuration)
	} else {
		fmt.Printf("Error: %s\n", result.Error)
	}
}
```

## Advanced Examples

### Batch Processing

```python
import os
import requests
from concurrent.futures import ThreadPoolExecutor

def process_single(image_path):
    url = "http://localhost:8080/api/process-invoice"
    with open(image_path, "rb") as f:
        files = {"file": f}
        response = requests.post(url, files=files)
        return image_path, response.json()

def batch_process(image_dir):
    image_files = [
        os.path.join(image_dir, f)
        for f in os.listdir(image_dir)
        if f.lower().endswith(('.jpg', '.jpeg', '.png'))
    ]

    results = []
    with ThreadPoolExecutor(max_workers=5) as executor:
        for image_path, result in executor.map(process_single, image_files):
            results.append({
                "file": image_path,
                "result": result
            })

    return results

# Usage
results = batch_process("./invoices")
for r in results:
    if r["result"]["success"]:
        print(f"{r['file']}: ${r['result']['invoice']['total']}")
    else:
        print(f"{r['file']}: ERROR - {r['result']['error']}")
```

### Retry Logic

```python
import time
import requests

def process_with_retry(image_path, max_retries=3):
    url = "http://localhost:8080/api/process-invoice"

    for attempt in range(max_retries):
        try:
            with open(image_path, "rb") as f:
                files = {"file": f}
                response = requests.post(url, files=files, timeout=120)
                result = response.json()

                if result["success"]:
                    return result

                # If AI extraction failed, try vision model
                if attempt < max_retries - 1:
                    print(f"Attempt {attempt + 1} failed, trying vision model...")
                    with open(image_path, "rb") as f2:
                        files = {"file": f2}
                        data = {"useVisionModel": "true"}
                        response = requests.post(url, files=files, data=data, timeout=120)
                        result = response.json()

                        if result["success"]:
                            return result

        except requests.exceptions.Timeout:
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt  # Exponential backoff
                print(f"Timeout, retrying in {wait_time}s...")
                time.sleep(wait_time)
            else:
                return {"success": False, "error": "Maximum retries exceeded"}

        except Exception as e:
            return {"success": False, "error": str(e)}

    return {"success": False, "error": "Failed after all retries"}
```

### Integration with Database

```python
import requests
import sqlite3
from datetime import datetime

def save_invoice_to_db(invoice_data):
    conn = sqlite3.connect('invoices.db')
    cursor = conn.cursor()

    # Create table if not exists
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS invoices (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            vendor TEXT,
            date TEXT,
            total REAL,
            tax REAL,
            categories TEXT,
            raw_text TEXT,
            processed_at TEXT,
            confidence REAL
        )
    ''')

    # Insert invoice
    cursor.execute('''
        INSERT INTO invoices (vendor, date, total, tax, categories, raw_text, processed_at, confidence)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    ''', (
        invoice_data['vendor'],
        invoice_data['date'],
        invoice_data['total'],
        invoice_data.get('tax', 0),
        ','.join(invoice_data.get('categories', [])),
        invoice_data.get('rawText', ''),
        invoice_data['processedAt'],
        invoice_data['confidence']
    ))

    invoice_id = cursor.lastrowid

    # Insert items
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS invoice_items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            invoice_id INTEGER,
            name TEXT,
            amount REAL,
            is_taxed BOOLEAN,
            quantity INTEGER,
            FOREIGN KEY (invoice_id) REFERENCES invoices (id)
        )
    ''')

    for item in invoice_data.get('items', []):
        cursor.execute('''
            INSERT INTO invoice_items (invoice_id, name, amount, is_taxed, quantity)
            VALUES (?, ?, ?, ?, ?)
        ''', (
            invoice_id,
            item['name'],
            item['amount'],
            item['isTaxed'],
            item.get('quantity', 1)
        ))

    conn.commit()
    conn.close()

    return invoice_id

# Usage
def process_and_save(image_path):
    url = "http://localhost:8080/api/process-invoice"

    with open(image_path, "rb") as f:
        files = {"file": f}
        response = requests.post(url, files=files)
        result = response.json()

        if result["success"]:
            invoice_id = save_invoice_to_db(result["invoice"])
            print(f"Invoice saved with ID: {invoice_id}")
            return invoice_id
        else:
            print(f"Error: {result['error']}")
            return None
```

### Webhook Integration

```python
import requests

def process_with_webhook(image_path, webhook_url):
    """Process invoice and send result to webhook"""

    # Process invoice
    url = "http://localhost:8080/api/process-invoice"
    with open(image_path, "rb") as f:
        files = {"file": f}
        response = requests.post(url, files=files)
        result = response.json()

    # Send to webhook
    if result["success"]:
        webhook_data = {
            "event": "invoice.processed",
            "invoice": result["invoice"],
            "metadata": {
                "ocrDuration": result.get("ocrDuration", 0),
                "aiDuration": result.get("aiDuration", 0),
                "totalDuration": result["totalDuration"]
            }
        }
    else:
        webhook_data = {
            "event": "invoice.failed",
            "error": result["error"]
        }

    requests.post(webhook_url, json=webhook_data)
    return result

# Usage
result = process_with_webhook(
    "invoice.jpg",
    "https://myapp.com/webhooks/invoice"
)
```

## Testing

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "time": "2024-01-15T14:30:00Z"
}
```

### Test with Sample Image

```bash
# Download sample receipt
curl -o sample.jpg https://example.com/sample-receipt.jpg

# Process it
curl -X POST http://localhost:8080/api/process-invoice \
  -F "file=@sample.jpg" \
  | jq .
```

## Performance Testing

### Load Test with Apache Bench

```bash
# Create test script
cat > test.sh << 'EOF'
#!/bin/bash
curl -X POST http://localhost:8080/api/process-invoice \
  -F "file=@invoice.jpg" \
  -w "%{time_total}\n" \
  -o /dev/null \
  -s
EOF

chmod +x test.sh

# Run 100 requests with 10 concurrent
seq 1 100 | xargs -n1 -P10 ./test.sh | \
  awk '{sum+=$1; count++} END {print "Average:", sum/count, "seconds"}'
```

### Load Test with Vegeta

```bash
# Install vegeta
go install github.com/tsenart/vegeta@latest

# Create target file
echo "POST http://localhost:8080/api/process-invoice" > targets.txt

# Run load test
cat targets.txt | vegeta attack -duration=30s -rate=10 | \
  vegeta report -type=text
```

## Monitoring

### Prometheus Metrics (Future)

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'invoice-ocr'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Custom Logging

```python
import logging
import requests

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def process_with_logging(image_path):
    logger.info(f"Processing {image_path}")

    url = "http://localhost:8080/api/process-invoice"
    with open(image_path, "rb") as f:
        files = {"file": f}
        response = requests.post(url, files=files)
        result = response.json()

        if result["success"]:
            logger.info(
                f"Success: vendor={result['invoice']['vendor']}, "
                f"total={result['invoice']['total']}, "
                f"duration={result['totalDuration']:.2f}s"
            )
        else:
            logger.error(f"Failed: {result['error']}")

        return result
```
