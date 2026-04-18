package pkg

import (
    "example.com/recursivetest/pkg/models"
    "example.com/recursivetest/pkg/api"
    "example.com/recursivetest/pkg/utils"
)

// RootConfig is the root configuration struct
// Contains nested references to sub-packages
type RootConfig struct {
    Name    string
    Enabled bool
    Timeout int
    User    *models.UserModel
    Handler *api.Handler
    Helper  *utils.Helper
}

// ConfigWithNested has embedded struct
type ConfigWithNested struct {
    ID      int64
    Data    string
    Enabled bool
}
