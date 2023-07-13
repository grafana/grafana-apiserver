print("""
                    ▓███▄
                  ▐███████
              ▄▄████████████████████
    ▐███████████████████████████████
    ▐█████████████▀▀          ▀▀█████▄
     ██████████▀                  ▀███▌
      ████████          ▄▄▄▄▄        ███
      ▐█████▌       ▄▓██████████▓▄    ██▌
      ██████       ████▀▀  ▀▀██████▄   ██
    ▄██████▌      ███          ▀█████
  ▓████████▌      ██             ████▌
 ███████████      ██▌            █████
 ▀██████████▌      ▀██▄          █████▄
    ▀█████████        ▀▀▀▀      ███████▌
        ▀███████▄             ▄████████▀
          █████████▄▄▄▄  ▄▄▄███████▀
          ███████████████████████▀
          ██████████████████████
          ▐██▀▀▀        ▀██████
                             ▀▀
""")


local("deploy/kind-with-registry.sh || true")

os = str(local('uname -s')).strip().lower()

local_resource(
  'grafana-apiserver',
  'go build -gcflags "all=-N -l" -o build/grafana-apiserver .',
  deps=['./main.go', './pkg'],
  serve_cmd='./build/grafana-apiserver --secure-port 8443 --kubeconfig ~/.kube/config --authentication-kubeconfig ~/.kube/config --authorization-kubeconfig ~/.kube/config -v 8',
)

k8s_yaml(kustomize('deploy/local-%s' % os))
