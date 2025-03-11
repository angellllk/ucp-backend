package config

import (
	"errors"
	"github.com/Jeffail/gabs/v2"
)

type Config struct {
	Version      string `json:"version"`
	FEPath       string `json:"frontend_path"`
	Dsn          string `json:"dsn"`
	Port         string `json:"port"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`
}

func Read(path string) (*Config, error) {
	parsed, err := gabs.ParseJSONFile(path)
	if err != nil {
		return nil, err
	}

	dsn, ok := parsed.Path("db.dsn").Data().(string)
	if !ok {
		return nil, errors.New("error dsn cast to string")
	}

	port, ok := parsed.Path("port").Data().(string)
	if !ok {
		return nil, errors.New("error port cast to string")
	}

	fe, ok := parsed.Path("frontend_path").Data().(string)
	if !ok {
		return nil, errors.New("error log cast to string")
	}

	version, ok := parsed.Path("version").Data().(string)
	if !ok {
		return nil, errors.New("error version cast to string")
	}

	smtpHost, ok := parsed.Path("email.smtp_host").Data().(string)
	if !ok {
		return nil, errors.New("error smtp host cast to string")
	}

	smtpPort, ok := parsed.Path("email.smtp_port").Data().(float64)
	if !ok {
		return nil, errors.New("error smtp port cast to string")
	}

	smtpUser, ok := parsed.Path("email.smtp_user").Data().(string)
	if !ok {
		return nil, errors.New("error smtp user cast to string")
	}

	smtpPwd, ok := parsed.Path("email.smtp_password").Data().(string)
	if !ok {
		return nil, errors.New("error smtp password cast to string")
	}

	smtpFrom, ok := parsed.Path("email.smtp_from").Data().(string)
	if !ok {
		return nil, errors.New("error smtp from cast to string")
	}

	return &Config{
		Dsn:          dsn,
		Port:         port,
		FEPath:       fe,
		Version:      version,
		SMTPHost:     smtpHost,
		SMTPPort:     int(smtpPort),
		SMTPUser:     smtpUser,
		SMTPPassword: smtpPwd,
		SMTPFrom:     smtpFrom,
	}, nil
}
