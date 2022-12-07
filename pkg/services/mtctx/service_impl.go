package mtctx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/appcontext"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/services/sqlstore/session"
	"github.com/grafana/grafana/pkg/setting"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type serviceImpl struct {
	clientset *kubernetes.Clientset
	cache     map[int64]*TenantInfo
	//watchers  map[int64]watch.Interface
}

// POC: shows stackID of tenant in current request
func (s *serviceImpl) showTenantInfo(c *models.ReqContext) response.Response {
	t, err := TenantInfoFromContext(c.Req.Context())
	if err != nil {
		fmt.Println("Could not find tenant info", err)
		return response.JSON(500, map[string]interface{}{
			"nope": "nope",
			"user": c.SignedInUser,
			"env":  os.Getenv("HG_STACK_ID"),
		})
	}

	return response.JSON(200, map[string]interface{}{
		"stackID": t.StackID,
		"user":    c.SignedInUser,
		"env":     os.Getenv("HG_STACK_ID"),
	})
}

// Gets config.ini from kubernetes and returns a watcher to listen for changes
func (s *serviceImpl) GetStackConfigWatcher(ctx context.Context, stackID int64) (watch.Interface, error) {
	if s.clientset == nil {
		return nil, fmt.Errorf("missing error")
	}

	return s.clientset.CoreV1().ConfigMaps("hosted-grafana").Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata[name]=%s", stackName(stackID)),
	})
}

// MIDDLEWARE: Adds TenantInfo
func (s *serviceImpl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// only run if on api
		if !strings.HasPrefix(r.RequestURI, "/api") {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		user, err := appcontext.User(ctx)
		if err != nil {
			fmt.Println("missing user", err)
			next.ServeHTTP(w, r)
			return
		}

		// Check cache for config by stackID
		info, ok := s.cache[user.StackID]

		// If no config, get one
		if !ok {
			var config *v1.ConfigMap

			// get the initial config map
			if s.clientset != nil {
				config, err = s.clientset.CoreV1().ConfigMaps("hosted-grafana").Get(context.TODO(), stackName(user.StackID), metav1.GetOptions{})
				if err != nil {
					fmt.Println("Error getting config map:", err)
				}
			}

			// build tenant info and add to context
			info = buildTenantInfoFromConfig(user.StackID, config)
			fmt.Println("POTATO: context set")
			s.cache[user.StackID] = info

			// Get config watcher
			//w, err := s.GetStackConfigWatcher(context.TODO(), stackID)
			//if err != nil {
			//fmt.Println("Error getting watcher for stackID:", stackID)
			//}

			// TODO should we check to see if we already have a watcher?
			// Also, we don't currently have a scenario where we remove a watcher, but we
			// should think through this.
			//if cachedWatcher, ok := s.watchers[stackID]; ok {
			//  // this should never happen

			//  cachedWatcher.Stop() // make sure we stop listening
			//  fmt.Println("WARNING: we found a watcher for a tenant that was missing tenantInfo")
			//}

			//// queue watcher
			//s.watchers[stackID] = w
		}

		ctx = ContextWithTenantInfo(ctx, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TODO: Finish me. Don't need for demo.
// Watches config maps for changes to tenant configs and removing tenants.
// Context is so we can cancel this blocking call, not a web context
//func (s *serviceImpl) WatchConfigMaps(ctx context.Context) {
//  agg := make(chan string)

//  // collect channels from all watchers
//  var chans []channel
//  for _, c := range s.watchers {
//    chans := append(chans, c)
//  }

//  // all config updates are handled the same. aggregaggregate them into a single channel we can use to
//  // receive
//  for _, ch := range chans {
//    go func(c chan string) {
//      for msg := range c {
//        agg <- msg
//      }
//    }(ch)
//  }

//  select {
//  case msg <- agg:
//    fmt.Println("received ", msg)
//  }
//}

// Builds the kubernetes stackname
func stackName(stackID int64) string {
	return fmt.Sprintf("%d-mt-config", stackID)
}

// takes INI, initializes db connection and returns it
func initializeDBConnection(config *v1.ConfigMap) *session.SessionDB {
	if config == nil {
		fmt.Printf("missing config!")
		return nil
	}
	jsontxt, ok := config.Data["ini"]
	if !ok {
		fmt.Printf("could not find key ini")
		return nil
	}

	jsonmap := make(map[string]map[string]string, 0)
	err := json.Unmarshal([]byte(jsontxt), &jsonmap)
	if err != nil {
		fmt.Printf("missing config!")
		return nil
	}
	cfg, err := setting.FromJSON(jsonmap)
	if err != nil {
		fmt.Printf("error reading json config: " + err.Error())
		return nil
	}

	// a whold new instance?   ┗|・o・|┛
	ss, err := sqlstore.NewSQLStore(cfg, nil, nil, nil, nil, nil)
	if err != nil {
		fmt.Printf("error reading json config: " + err.Error())
		return nil
	}

	return ss.GetSqlxSession()
}

// Builds tenant info and returns it to enter into cache
func buildTenantInfoFromConfig(stackID int64, config *v1.ConfigMap) *TenantInfo {
	dbCon := initializeDBConnection(config)

	return &TenantInfo{
		StackID:   stackID,
		SessionDB: dbCon,
	}
}