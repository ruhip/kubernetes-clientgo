package k8s

var DefaultK8sMgr *K8sManager

func NewK8sManager() error {
	var err error
	if err = InitK8sClinet(); err != nil {
		return err
	}
	DefaultK8sMgr, err = NewManager()
	return err
}
