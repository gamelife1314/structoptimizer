package api

// Handler is the API handler
type Handler struct {
    Timeout int
    Name    string
}

// UserService handles user operations
type UserService struct {
    Cache map[int64]string
    Count int
}
