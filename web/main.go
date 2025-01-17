package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/suggestions", suggestionsHandler)
	http.Handle("/", http.FileServer(http.Dir("."))) // Serve the HTML, CSS, and JS files
	fmt.Println("Server is running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	suggestions := []string{"Apple", "Banana", "Cherry", "Kokos", "orange", "baklagan", "pomidor", "KAVUN"} // Replace with your logic
	fmt.Println("##############################################")
	if q := strings.Split(query, " "); len(q) > 1 {
		query = q[len(q)-1]
	}
	for _, suggestion := range suggestions {
		// if strings.Contains(strings.ToLower(suggestion), strings.ToLower(query)) {
		if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(query)) {
			fmt.Printf("<div class=\"suggestion-item\">%s</div>\n", suggestion)

			fmt.Fprintf(w, "<div class=\"suggestion-item\">%s</div>", suggestion)
		}
	}
}
