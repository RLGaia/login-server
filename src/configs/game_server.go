package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const EnvServerIpKey = "SERVER_IP"
const EnvServerLocationKey = "SERVER_LOCATION"
const EnvServerNameKey = "SERVER_NAME"
const EnvServerPortKey = "SERVER_PORT"
const EnvServerPathKey = "SERVER_PATH"
const EnvServerPvpTypeKey = "SERVER_PVP_TYPE"
const DefaultServerName = "OTServBR-Global"

const (
	ConfigErrorCodeUnknown              = 1000
	ConfigErrorCodeServerNameMismatch   = 1001
	ConfigErrorCodeServerConfigInvalid  = 1002
	ConfigErrorCodeServerConfigNotFound = 1003

	ConfigErrorUnknown              = "UNKNOWN_CONFIG_ERROR"
	ConfigErrorServerConfigNotFound = "SERVER_CONFIG_NOT_FOUND"
	ConfigErrorServerConfigInvalid  = "SERVER_CONFIG_INVALID"
	ConfigErrorServerNameMismatch   = "SERVER_NAME_MISMATCH"
)

type ConfigurationError struct {
	Code    int
	Name    string
	Message string
	Cause   error
}

func (err *ConfigurationError) Error() string {
	if err == nil {
		return ""
	}
	if err.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", err.Name, err.Message, err.Cause)
	}
	return fmt.Sprintf("%s: %s", err.Name, err.Message)
}

func (err *ConfigurationError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

type GameServerConfigs struct {
	Port     int
	Name     string
	IP       string
	Location string
	Config
}

func (gameServerConfigs *GameServerConfigs) Format() string {
	return fmt.Sprintf(
		"Connected with %s server %s:%d - %s",
		gameServerConfigs.Name,
		gameServerConfigs.IP,
		gameServerConfigs.Port,
		gameServerConfigs.Location,
	)
}

// PvP types understood by the client's character list.
const (
	PvpTypeOpen     uint32 = 0
	PvpTypeOptional uint32 = 1
	PvpTypeHardcore uint32 = 2
)

// PvpTypeFromWorldType maps Canary's worldType setting ("pvp", "no-pvp",
// "pvp-enforced") to the PvP type the client shows in its character list.
// Unknown or empty values are reported as open PvP, the previous behavior.
func PvpTypeFromWorldType(worldType string) uint32 {
	switch strings.ToLower(strings.TrimSpace(worldType)) {
	case "no-pvp":
		return PvpTypeOptional
	case "pvp-enforced":
		return PvpTypeHardcore
	default:
		return PvpTypeOpen
	}
}

// ResolvePvpType returns SERVER_PVP_TYPE when it holds a valid number and
// otherwise the PvP type derived from Canary's worldType.
func ResolvePvpType(worldType string) uint32 {
	value := strings.TrimSpace(GetEnvStr(EnvServerPvpTypeKey, ""))
	if value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
			return uint32(parsed)
		}
	}
	return PvpTypeFromWorldType(worldType)
}

func GetGameServerConfigs() GameServerConfigs {
	return GameServerConfigs{
		IP:       GetEnvStr(EnvServerIpKey, "127.0.0.1"),
		Name:     GetEnvStr(EnvServerNameKey, DefaultServerName),
		Port:     GetEnvInt(EnvServerPortKey, 7172),
		Location: GetEnvStr(EnvServerLocationKey, "BRA"),
	}
}

func ValidateGameServerName(gameConfigs GameServerConfigs) error {
	serverPath := strings.TrimSpace(GetEnvStr(EnvServerPathKey, ""))
	if serverPath == "" {
		return nil
	}

	configPath, err := findServerConfigPath(serverPath)
	if err != nil {
		return err
	}

	manager, err := NewLuaConfigManager(configPath)
	if err != nil {
		return &ConfigurationError{
			Code:    ConfigErrorCodeServerConfigInvalid,
			Name:    ConfigErrorServerConfigInvalid,
			Message: fmt.Sprintf("failed to load Canary config from %s", configPath),
			Cause:   err,
		}
	}

	canaryServerName := strings.TrimSpace(manager.GetString("serverName"))
	if canaryServerName == "" {
		return &ConfigurationError{
			Code:    ConfigErrorCodeServerConfigInvalid,
			Name:    ConfigErrorServerConfigInvalid,
			Message: fmt.Sprintf("Canary config %s does not define serverName", configPath),
		}
	}

	if canaryServerName == strings.TrimSpace(gameConfigs.Name) {
		return nil
	}

	return &ConfigurationError{
		Code: ConfigErrorCodeServerNameMismatch,
		Name: ConfigErrorServerNameMismatch,
		Message: fmt.Sprintf(
			"login-server SERVER_NAME=%q but Canary config.lua serverName=%q",
			gameConfigs.Name,
			canaryServerName,
		),
	}
}

func findServerConfigPath(serverPath string) (string, error) {
	configPath := filepath.Join(serverPath, "config.lua")
	if exists, err := configFileExists(configPath); err != nil {
		return "", err
	} else if exists {
		return configPath, nil
	}

	distPath := filepath.Join(serverPath, "config.lua.dist")
	if exists, err := configFileExists(distPath); err != nil {
		return "", err
	} else if exists {
		return distPath, nil
	}

	return "", &ConfigurationError{
		Code:    ConfigErrorCodeServerConfigNotFound,
		Name:    ConfigErrorServerConfigNotFound,
		Message: "SERVER_PATH does not contain config.lua or config.lua.dist",
	}
}

func configFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return false, &ConfigurationError{
				Code:    ConfigErrorCodeServerConfigInvalid,
				Name:    ConfigErrorServerConfigInvalid,
				Message: fmt.Sprintf("%s is a directory, expected a config file", path),
			}
		}
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, &ConfigurationError{
		Code:    ConfigErrorCodeServerConfigInvalid,
		Name:    ConfigErrorServerConfigInvalid,
		Message: fmt.Sprintf("failed to inspect Canary config path %s", path),
		Cause:   err,
	}
}

const EnvVocations = "VOCATIONS"

func GetServerVocations() []string {
	vocationsStr := GetEnvStr(EnvVocations, "")
	vocations := strings.Split(vocationsStr, ",")

	if len(vocationsStr) == 0 || len(vocations) == 0 {
		return []string{
			"None",
			"Sorcerer",
			"Druid",
			"Paladin",
			"Knight",
			"Master Sorcerer",
			"Elder Druid",
			"Royal Paladin",
			"Elite Knight",
			"Sorcerer Dawnport",
			"Druid Dawnport",
			"Paladin Dawnport",
			"Knight Dawnport",
		}
	}

	return vocations
}
