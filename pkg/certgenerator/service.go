package certgenerator

import (
	"context"
	"fmt"
	"github.com/grafana/dskit/services"
)

const (
	DefaultAPIServerIp = "127.0.0.1"
)

var (
	_ ServiceInterface = (*Service)(nil)
)

type ServiceInterface interface {
	services.NamedService
}

type Service struct {
	*services.BasicService
	certUtil *CertUtil
}

func CreateService(serviceName string, k8sDataPath string) (*Service, error) {
	certUtil := &CertUtil{
		K8sDataPath: k8sDataPath,
	}

	s := &Service{
		certUtil: certUtil,
	}

	s.BasicService = services.NewIdleService(s.up, nil).WithName(serviceName)

	return s, nil
}
func (s *Service) up(ctx context.Context) error {
	err := s.certUtil.InitializeCACertPKI()
	if err != nil {
		fmt.Printf("error initializing CA", "error", err.Error())
		return err
	}

	err = s.certUtil.EnsureApiServerPKI(DefaultAPIServerIp)
	if err != nil {
		fmt.Printf("error ensuring API Server PKI", "error", err)
		return err
	}

	err = s.certUtil.EnsureAuthzClientPKI()
	if err != nil {
		fmt.Printf("error ensuring K8s Authz Client PKI", "error", err)
		return err
	}

	err = s.certUtil.EnsureAuthnClientPKI()
	if err != nil {
		fmt.Printf("error ensuring K8s Authn Client PKI", "error", err)
		return err
	}

	return nil
}
