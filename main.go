// package main

// import (
// 	"database/sql"
// 	"fmt"
// 	"log"

// 	_ "github.com/lib/pq"
// )

// func main() {
// 	connStr := "host=localhost port=5432 user=postgres password=rx dbname=todo sslmode=disable"
// 	db, err := sql.Open("postgres", connStr)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer db.Close()

// 	err = db.Ping()
// 	if err != nil {
// 		log.Fatal("Failed to connect to the database:", err)
// 	}

// 	fmt.Println("Connected to the database successfully!")
// 	getUsers(db)
// }

// func getUsers(db *sql.DB) {
// 	//rows, err := db.Query("SELECT id, description FROM tasks")
// 	rows, err := db.Query(`SELECT id, description FROM "Tasks"`)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var id int
// 		var name string
// 		if err := rows.Scan(&id, &name); err != nil {
// 			log.Fatal(err)
// 		}
// 		fmt.Printf("User ID: %d, Name: %s\n", id, name)
// 	}
// }

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

type task struct {
	Id   int    `json:"id"`
	Desc string `json:"desc"`
}

func main() {
	connStr := "host=localhost port=5432 user=postgres password=rx dbname=todo sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	fmt.Println("Connected to the database successfully!")
	http.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			add(w, r, db)
		case http.MethodGet:
			list(w, db)
		case http.MethodPut:
			update(w, r, db)
		case http.MethodDelete:
			delete(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
func add(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var newTask task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil || newTask.Desc == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	var id int
	p, err := db.Query(`SELECT COALESCE(MIN(t1.id + 1), 1) AS missing_id FROM "Tasks" t1 LEFT JOIN "Tasks" t2 ON t1.id + 1 = t2.id WHERE t2.id IS NULL`)
	if err != nil {
		log.Fatal(err)
	}
	p.Next()
	if err := p.Scan(&id); err != nil {
		log.Fatal(err)
	}

	result, err := db.Exec(`INSERT INTO "Tasks" (id,description) VALUES ($1,$2)`, id, newTask.Desc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	newTask.Id = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newTask)
}
func list(w http.ResponseWriter, db *sql.DB) {
	//rows, err := db.Query("SELECT id, description FROM tasks")
	count, err := db.Query(`SELECT COUNT(id) FROM "Tasks"`)
	if err != nil {
		log.Fatal(err)
	}
	defer count.Close()
	var c int
	count.Next()
	if err := count.Scan(&c); err != nil {
		log.Fatal(err)
	}
	if c == 0 {
		//w.WriteHeader(http.StatusNoContent)
		//http.Error(w, "", http.StatusNoContent)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "No tasks found", "count": c, "tasks": []task{}})
		return
	}
	rows, err := db.Query(`SELECT id, description FROM "Tasks" ORDER BY id ASC`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var tasks []task
	for rows.Next() {
		var id int
		var desc string
		if err := rows.Scan(&id, &desc); err != nil {
			log.Fatal(err)
			return
		}
		var t task
		t.Id = id
		t.Desc = desc
		tasks = append(tasks, t)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//fmt.Fprintln(w, "TASK ID: ", id, "DESCRIPTION: ", desc)
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks, "message": "success", "count": c})
}
func update(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var newTask task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil || newTask.Id == 0 || newTask.Desc == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`UPDATE "Tasks" SET description=$2 WHERE id=$1`, newTask.Id, newTask.Desc)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(result)
	RowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	if RowsAffected == 0 {
		http.Error(w, "Invalid Request", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newTask)
}
func delete(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var newTask task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil || newTask.Id == 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM "Tasks" WHERE id=$1`, newTask.Id)
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	RowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Fatal(err)
		return
	}
	if RowsAffected == 0 {
		// http.Error(w, "Invalid Request", http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Task not found!"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task has been deleted successfully!"})
}
