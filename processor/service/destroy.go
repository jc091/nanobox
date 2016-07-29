package service

import (
	"fmt"
	"net"
	"strings"

	"github.com/jcelliott/lumber"
	"github.com/nanobox-io/golang-docker-client"
	"github.com/nanobox-io/nanobox-golang-stylish"

	"github.com/nanobox-io/nanobox/models"
	"github.com/nanobox-io/nanobox/processor"
	"github.com/nanobox-io/nanobox/provider"
	"github.com/nanobox-io/nanobox/util/config"
	"github.com/nanobox-io/nanobox/util/data"
	"github.com/nanobox-io/nanobox/util/dhcp"
)

// processServiceDestroy ...
type processServiceDestroy struct {
	control processor.ProcessControl
	app     models.App
	service models.Service
}

//
func init() {
	processor.Register("service_destroy", serviceDestroyFn)
}

//
func serviceDestroyFn(control processor.ProcessControl) (processor.Processor, error) {
	// make sure we have the name of the service before we continue
	if control.Meta["name"] == "" {
		return nil, fmt.Errorf("missing image or name")
	}
	if control.Meta["app_name"] == "" {
		control.Meta["app_name"] = fmt.Sprintf("%s_%s", config.AppID(), control.Env)
	}
	return processServiceDestroy{control: control}, nil
}

//
func (serviceDestroy processServiceDestroy) Results() processor.ProcessControl {
	return serviceDestroy.control
}

//
func (serviceDestroy processServiceDestroy) Process() error {

	if err := serviceDestroy.loadApp(); err != nil {
		return err
	}

	if err := serviceDestroy.loadService(); err != nil {
		return err
	}

	if err := serviceDestroy.printDisplay(); err != nil {
		return err
	}

	if err := serviceDestroy.removeContainer(); err != nil {
		// if im unable to remove the container (especially if it doesnt exist)
		// we shouldnt fail
		lumber.Error("unable to removeContainer: %s", err.Error())
	}

	if err := serviceDestroy.detachNetwork(); err != nil {
		lumber.Error("unable to detachNetwork: %s", err.Error())
		return err
	}

	if err := serviceDestroy.removeEvars(); err != nil {
		lumber.Error("unable to removeEvars: %s", err.Error())
		return err
	}

	if err := serviceDestroy.deleteService(); err != nil {
		lumber.Error("unable to deleteService: %s", err.Error())
		return err
	}

	return nil
}

// loadApp loads the app from the database
func (serviceDestroy *processServiceDestroy) loadApp() error {

	// load the app from the database
	return data.Get("apps", serviceDestroy.control.Meta["app_name"], &serviceDestroy.app)
}

// loadService fetches the service from the database
func (serviceDestroy *processServiceDestroy) loadService() error {
	name := serviceDestroy.control.Meta["name"]

	data.Get(serviceDestroy.control.Meta["app_name"], name, &serviceDestroy.service)

	return nil
}

// printDisplay prints the user display for progress
func (serviceDestroy *processServiceDestroy) printDisplay() error {

	name := serviceDestroy.control.Meta["name"]
	message := stylish.Bullet("Destroying %s", name)

	// print!
	serviceDestroy.control.Display(message)

	return nil
}

// removeContainer destroys the docker container
func (serviceDestroy *processServiceDestroy) removeContainer() error {

	name := serviceDestroy.control.Meta["name"]
	container := fmt.Sprintf("nanobox_%s_%s", serviceDestroy.control.Meta["app_name"], name)

	if err := docker.ContainerRemove(container); err != nil {
		return err
	}

	return nil
}

// detachNetwork detaches the virtual network from the host
func (serviceDestroy *processServiceDestroy) detachNetwork() error {

	name := serviceDestroy.control.Meta["name"]
	service := serviceDestroy.service

	if err := provider.RemoveNat(service.ExternalIP, service.InternalIP); err != nil {
		return err
	}

	if err := provider.RemoveIP(service.ExternalIP); err != nil {
		return err
	}

	// don't return the external IP if this is portal
	if name != "portal" && serviceDestroy.app.GlobalIPs[name] == "" {
		if err := dhcp.ReturnIP(net.ParseIP(service.ExternalIP)); err != nil {
			return err
		}
	}

	// don't return the internal IP if it's an app-level cache
	if serviceDestroy.app.LocalIPs[name] == "" {
		if err := dhcp.ReturnIP(net.ParseIP(service.InternalIP)); err != nil {
			return err
		}
	}

	return nil
}

// removeEvars removes any env vars associated with this service
func (serviceDestroy processServiceDestroy) removeEvars() error {
	// fetch the environment variables model
	envVars := models.Evars{}
	data.Get(config.AppID()+"_meta", "env", &envVars)

	// create a prefix for each of the environment variables.
	// for example, if the service is 'data.db' the prefix
	// would be DATA_DB. Dots are replaced with underscores,
	// and characters are uppercased.
	name := serviceDestroy.control.Meta["name"]
	prefix := strings.ToUpper(strings.Replace(name, ".", "_", -1))

	// we loop over all environment variables and see if the key contains
	// the prefix above. If so, we delete the item.
	for key := range envVars {
		if strings.HasPrefix(key, prefix) {
			delete(envVars, key)
		}
	}

	// persist the evars
	if err := data.Put(config.AppID()+"_meta", "env", envVars); err != nil {
		return err
	}

	return nil
}

// deleteService deletes the service record from the db
func (serviceDestroy processServiceDestroy) deleteService() error {

	name := serviceDestroy.control.Meta["name"]

	if err := data.Delete(serviceDestroy.control.Meta["app_name"], name); err != nil {
		return err
	}

	return nil
}
