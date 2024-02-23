package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Order struct {
	ID                   int         `json:"id"`
	Items                []OrderItem `json:"items"`
	Total                float64     `json:"total"`
	ReceiptName          string      `json:"receipt_name"`
	IdentificationNumber string      `json:"identification_number"`
	PhoneNumber          string      `json:"phone_number"`
	Email                string      `json:"email"`
	Address              string      `json:"address"`
	WorkshopAddress      string      `json:"workshop_address"`
	Date                 string      `json:"date"`
	Hour                 string      `json:"hour"`
}

type OrderItem struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

var db *sql.DB

func init() {
	// Initialize the database connection in an init function
	var err error
	db, err = sql.Open("postgres", "postgres://tavito:mamacita@localhost:5432/orders?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	// Check the database connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping the database:", err)
	}
}

func main() {
	defer db.Close()

	router := mux.NewRouter()

	// Add CORS middleware to the router
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:3000", "http://localhost:8080"}), // Replace "*" with your allowed origins
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),        // Include OPTIONS method
		handlers.AllowedHeaders([]string{"Content-Type"}),
	)

	// Use the CORS middleware with your router
	router.Use(corsMiddleware)

	//Endpoints to manage orders
	router.HandleFunc("/orders", createOrder).Methods("POST", "OPTIONS") // Handle OPTIONS for preflight
	router.HandleFunc("/orders", GetOrders).Methods("GET")               // endpoint to get all orders
	router.HandleFunc("/orders/{id}", deleteOrder).Methods("DELETE", "OPTIONS")
	http.ListenAndServe(":3000", router)
}

// Endpoint to get all orders
func GetOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Execute a SELECT query to retrieve all orders from the database
	rows, err := db.Query("SELECT * FROM orders")
	if err != nil {
		http.Error(w, "Failed to retrieve orders from the database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Create a slice to store the retrieved orders
	var retrievedOrders []Order

	// Iterate through the result rows and scan them into Order structs
	for rows.Next() {
		var order Order
		if err := rows.Scan(
			&order.ID,
			&order.Total,
			&order.ReceiptName,
			&order.IdentificationNumber,
			&order.PhoneNumber,
			&order.Email,
			&order.Address,
			&order.WorkshopAddress,
			&order.Date,
			&order.Hour,
		); err != nil {
			http.Error(w, "Failed to scan order data from the database", http.StatusInternalServerError)
			return
		}

		// Fetch order items for the current order
		order.Items, err = getOrderItems(order.ID)
		if err != nil {
			http.Error(w, "Failed to retrieve order items from the database", http.StatusInternalServerError)
			return
		}

		retrievedOrders = append(retrievedOrders, order)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		http.Error(w, "Error while iterating over database rows", http.StatusInternalServerError)
		return
	}

	// Encode and send the retrieved orders as JSON response
	json.NewEncoder(w).Encode(retrievedOrders)
}

// Function to retrieve order items for a given order ID
func getOrderItems(orderID int) ([]OrderItem, error) {
	// Execute a SELECT query to retrieve order items for a specific order from the database
	rows, err := db.Query("SELECT name, price FROM order_items WHERE order_id = $1", orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create a slice to store the retrieved order items
	var orderItems []OrderItem

	// Iterate through the result rows and scan them into OrderItem structs
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.Name,
			&item.Price,
		); err != nil {
			return nil, err
		}
		orderItems = append(orderItems, item)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orderItems, nil
}

// Endpoint to receive orders
func createOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Insert order into the database
	orderID, err := insertOrder(order)
	if err != nil {
		http.Error(w, "Failed to insert order into the database", http.StatusInternalServerError)
		return
	}

	// Insert order items into the database
	if err := insertOrderItems(orderID, order.Items); err != nil {
		http.Error(w, "Failed to insert order items into the database", http.StatusInternalServerError)
		return
	}

	// Respond with a success message
	successMessage := map[string]string{"message": "Order received successfully"}
	json.NewEncoder(w).Encode(successMessage)
}

// Function to insert an order into the database
func insertOrder(order Order) (int, error) {
	var orderID int

	// Perform the SQL INSERT for orders table
	err := db.QueryRow(
		"INSERT INTO orders (total, receipt_name, identification_number, phone_number, email, address, workshop_address, date, hour) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
		order.Total, order.ReceiptName, order.IdentificationNumber, order.PhoneNumber, order.Email,
		order.Address, order.WorkshopAddress, order.Date, order.Hour,
	).Scan(&orderID)

	return orderID, err
}

// Function to insert order items into the database
func insertOrderItems(orderID int, items []OrderItem) error {
	// Perform the SQL INSERT for order_items table for each item
	for _, item := range items {
		_, err := db.Exec(
			"INSERT INTO order_items (order_id, name, price) VALUES ($1, $2, $3)",
			orderID, item.Name, item.Price,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// Endpoint to delete an order
func deleteOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the order ID from the request URL parameters
	params := mux.Vars(r)
	orderID := params["id"]

	// Delete order items associated with the order ID
	_, err := db.Exec("DELETE FROM order_items WHERE order_id = $1", orderID)
	if err != nil {
		log.Println("Error deleting order items:", err)
		http.Error(w, "Failed to delete order items from the database", http.StatusInternalServerError)
		return
	}

	// Execute the SQL DELETE statement to delete the order
	result, err := db.Exec("DELETE FROM orders WHERE id = $1", orderID)
	if err != nil {
		log.Println("Error deleting order:", err)
		http.Error(w, "Failed to delete order from the database", http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Error getting rows affected:", err)
		http.Error(w, "Failed to get the number of rows affected", http.StatusInternalServerError)
		return
	}

	// Respond with a success message
	if rowsAffected == 0 {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	successMessage := map[string]string{"message": "Order deleted successfully"}
	json.NewEncoder(w).Encode(successMessage)
}

/*
/////////////////////CURL COMMANDS TO THE TEST THE ENDPOINTS//////////////////////////////

curl -X POST -H "Content-Type: application/json" -d '{
  "items": [
    {"name": "Product1", "price": 20.0},
    {"name": "Product2", "price": 30.0}
  ],
  "total": 50.0,
  "receipt_name": "John Doe",
  "identification_number": "123456789",
  "phone_number": "+1234567890",
  "email": "john.doe@example.com",
  "address": "123 Main St, Cityville",
  "workshop_address": "456 Workshop St, Garage City",
  "date": "2023-10-25",
  "hour": "14:30"
}' http://localhost:8080/orders

curl -X DELETE http://localhost:3000/orders/{order_id}



/////////////////////DATA BASE CONFIGURATION IN DOCKER////////////////////////

# Build the PostgreSQL container
docker build -t autoparts -f Dockerfile .

# Run the PostgreSQL container
docker run -d -p 5432:5432 --name autoparts-container autoparts

/////////////////////POSTGRES DATABASE CONFIGURATION////////////////////////
# Init Postgres in bash
psql
# List databases
\l
# Create database
CREATE DATABASE orders;
# Switch to orders database
\c orders
# Check you path in UNIX bash
pwd
# Execute sql script
\i /path/to/orders.sql
# Delete database in case you need
DROP DATABASE orders;





*/
