package models

// UserModel represents a user
type UserModel struct {
    ID    int64
    Name  string
    Email string
}

// ProfileModel represents a user profile  
type ProfileModel struct {
    UserID int64
    Bio    string
    Avatar string
}
