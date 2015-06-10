package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mostlygeek/reaper/events"
)

// Reaper finds resources and deals with them
type Reaper struct {
	conf   Config
	errCh  chan error
	infoCh chan string
	runCh  chan bool
}

// NewReaper is a Reaper constructor shorthand
func NewReaper(c Config) *Reaper {
	return &Reaper{
		conf:   c,
		errCh:  make(chan error),
		infoCh: make(chan string),
		runCh:  make(chan bool),
	}
}

// Start begins Reaper execution in a new goroutine
func (r *Reaper) Start() {
	r.start()
}

// Stop closes a Reaper's stop channel
func (r *Reaper) Stop() {
	r.runCh <- false
}

// unexported start is continuous loop that reaps every
// time interval
func (r *Reaper) start() {
	// Log.Info("We started")
	// this needs to be a goroutine
	// race conditions?!
	go func() {
		r.runCh <- true
	}()
	for {
		// Log.Info("We forin'")
		select {
		case <-time.After(r.conf.Reaper.Interval.Duration):
			r.runCh <- true
		case run := <-r.runCh:
			// Log.Info("Running reaper ;)")
			if run {
				// run reaper
				go r.Once()
			} else {
				// exit!
				Log.Debug("Stopping reaper on runCh receiving false")
				close(r.runCh)
				return
			}
		case err := <-r.errCh:
			if err != nil {
				Log.Error("%s", err.Error())
			}
		case info := <-r.infoCh:
			Log.Info("%s", info)
		case <-time.After(time.Second * 1):
			r.infoCh <- "heartbeat"
		}
	}
}

// Once is run once every time interval by start
// it is intended to handle all reaping logic
func (r *Reaper) Once() {
	// run these as goroutines
	var reapFuncs = []func(chan bool){
		// r.reapInstances,
		// r.reapSecurityGroups,
		// r.reapVolumes,
		// r.reapSnapshots,
		// r.reapAutoScalingGroups,
		r.reap,
	}

	// we block execution waiting for done to fill
	// so that the "sleeping for X" message shows
	// after all reaping is completed
	done := make(chan bool, 1)
	for _, f := range reapFuncs {
		go f(done)
	}

	// TODO: I have no idea how concurrency works
	// TODO update: I have some idea of how concurrency works
	for i := 0; i < len(reapFuncs); i++ {
		<-done
	}

	if Conf.StateFile != "" {
		r.SaveState(Conf.StateFile)
	}

	Log.Notice("Sleeping for %s", r.conf.Reaper.Interval.Duration.String())
}

func (r *Reaper) SaveState(stateFile string) {
	// open file RW, create it if it doesn't exist
	s, err := os.OpenFile(Conf.StateFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	defer func() { s.Close() }()
	if err != nil {
		r.errCh <- fmt.Errorf("Unable to create StateFile '%s'", Conf.StateFile)
	} else {
		r.infoCh <- fmt.Sprintf("States will be saved to %s", Conf.StateFile)
	}
	// save state to state file
	for region := range Reapables {
		for id := range Reapables[region] {
			s.Write([]byte(fmt.Sprintf("%s,%s,%s\n", region, id, Reapables[region][id].ReaperState().String())))
		}
	}
}

// convenience function that returns a map of instances in ASGs
func allASGInstanceIds(as []AutoScalingGroup) map[string]map[string]bool {
	// maps region to id to bool
	inASG := make(map[string]map[string]bool)
	for _, region := range Conf.AWS.Regions {
		inASG[region] = make(map[string]bool)
	}
	for _, a := range as {
		for _, instance := range a.Instances {
			// add the instance to the map
			inASG[a.Region][instance] = true
		}
	}
	return inASG
}

// returns ASGs as filterables
func allAutoScalingGroups(errc chan error, infoc chan string) []Filterable {
	regions := Conf.AWS.Regions

	// waitgroup for goroutines
	var wg sync.WaitGroup

	// channel for creating SecurityGroups
	in := make(chan *AutoScalingGroup)

	for _, region := range regions {
		wg.Add(1)

		sum := 0

		// goroutine per region to fetch all security groups
		go func(region string) {
			defer wg.Done()
			api := autoscaling.New(&aws.Config{Region: region})

			// TODO: nextToken paging
			input := &autoscaling.DescribeAutoScalingGroupsInput{}
			resp, err := api.DescribeAutoScalingGroups(input)
			if err != nil {
				errc <- err
			}

			for _, a := range resp.AutoScalingGroups {
				sum += 1
				in <- NewAutoScalingGroup(region, a)
			}

			infoc <- fmt.Sprintf("Found %d total AutoScalingGroups in %s", sum, region)
			for _, e := range Events {
				go func(c chan error, e events.EventReporter) {
					c <- e.NewStatistic("reaper.asgs.total", float64(len(in)), []string{fmt.Sprintf("region:%s", region)})
				}(errc, e)
			}
		}(region)
	}
	// aggregate
	var autoScalingGroups []Filterable
	go func() {
		for a := range in {
			autoScalingGroups = append(autoScalingGroups, a)
		}
	}()

	// synchronous wait for all goroutines in wg to be done
	wg.Wait()

	// done with the channel
	close(in)

	infoc <- fmt.Sprintf("Found %d total ASGs.", len(autoScalingGroups))
	return autoScalingGroups
}

func (r *Reaper) reapSnapshots(done chan bool) {
	snapshots := allSnapshots()
	Log.Info(fmt.Sprintf("Total snapshots: %d", len(snapshots)))
	done <- true
}

func allSnapshots() []Filterable {
	regions := Conf.AWS.Regions

	// waitgroup for goroutines
	var wg sync.WaitGroup

	// channel for creating SecurityGroups
	in := make(chan *Snapshot)

	for _, region := range regions {
		wg.Add(1)

		sum := 0

		// goroutine per region to fetch all security groups
		go func(region string) {
			defer wg.Done()
			api := ec2.New(&aws.Config{Region: region})

			// TODO: nextToken paging
			input := &ec2.DescribeSnapshotsInput{}
			resp, err := api.DescribeSnapshots(input)
			if err != nil {
				// TODO: wee
			}

			for _, v := range resp.Snapshots {
				sum += 1
				in <- NewSnapshot(region, v)
			}

			Log.Info(fmt.Sprintf("Found %d total snapshots in %s", sum, region))
			for _, e := range Events {
				go e.NewStatistic("reaper.snapshots.total", float64(len(in)), []string{fmt.Sprintf("region:%s", region)})
			}
		}(region)
	}
	// aggregate
	var snapshots []Filterable
	go func() {
		for s := range in {
			// Reapables[s.Region][s.ID] = s
			snapshots = append(snapshots, s)
		}
	}()

	// synchronous wait for all goroutines in wg to be done
	wg.Wait()

	// done with the channel
	close(in)

	Log.Info("Found %d total snapshots.", len(snapshots))
	return snapshots
}

func (r *Reaper) reapVolumes(done chan bool) {
	volumes := allVolumes()
	r.infoCh <- fmt.Sprintf("Total volumes: %d", len(volumes))
	for _, e := range Events {
		e.NewStatistic("reaper.volumes.total", float64(len(volumes)), nil)
	}
	done <- true
}

func allVolumes() Volumes {
	regions := Conf.AWS.Regions

	// waitgroup for goroutines
	var wg sync.WaitGroup

	// channel for creating SecurityGroups
	in := make(chan *Volume)

	for _, region := range regions {
		wg.Add(1)

		sum := 0

		// goroutine per region to fetch all security groups
		go func(region string) {
			defer wg.Done()
			api := ec2.New(&aws.Config{Region: region})

			// TODO: nextToken paging
			input := &ec2.DescribeVolumesInput{}
			resp, err := api.DescribeVolumes(input)
			if err != nil {
				// TODO: wee
			}

			for _, v := range resp.Volumes {
				sum += 1
				in <- NewVolume(region, v)
			}

			Log.Info(fmt.Sprintf("Found %d total volumes in %s", sum, region))
		}(region)
	}
	// aggregate
	var volumes Volumes
	go func() {
		for v := range in {
			// Reapables[v.Region][v.ID] = v
			volumes = append(volumes, v)
		}
	}()

	// synchronous wait for all goroutines in wg to be done
	wg.Wait()

	// done with the channel
	close(in)

	Log.Info("Found %d total snapshots.", len(volumes))
	return volumes
}

func (r *Reaper) reapSecurityGroups(done chan bool) {
	securitygroups := allSecurityGroups()
	r.infoCh <- fmt.Sprintf("Total security groups: %d", len(securitygroups))
	for _, e := range Events {
		go e.NewStatistic("reaper.securitygroups.total", float64(len(securitygroups)), nil)
	}
	done <- true
}

func allSecurityGroups() SecurityGroups {
	regions := Conf.AWS.Regions

	// waitgroup for goroutines
	var wg sync.WaitGroup

	// channel for creating SecurityGroups
	in := make(chan *SecurityGroup)

	for _, region := range regions {
		wg.Add(1)

		sum := 0

		// goroutine per region to fetch all security groups
		go func(region string) {
			defer wg.Done()
			api := ec2.New(&aws.Config{Region: region})

			// TODO: nextToken paging
			input := &ec2.DescribeSecurityGroupsInput{}
			resp, err := api.DescribeSecurityGroups(input)
			if err != nil {
				// TODO: wee
			}

			for _, sg := range resp.SecurityGroups {
				sum += 1
				in <- NewSecurityGroup(region, sg)
			}

			Log.Info(fmt.Sprintf("Found %d total security groups in %s", sum, region))
		}(region)
	}
	// aggregate
	var securityGroups SecurityGroups
	go func() {
		for sg := range in {
			// Reapables[sg.Region][sg.ID] = sg
			securityGroups = append(securityGroups, sg)
		}
	}()

	// synchronous wait for all goroutines in wg to be done
	wg.Wait()

	// done with the channel
	close(in)

	Log.Info("Found %d total security groups.", len(securityGroups))
	return securityGroups
}

func (r *Reaper) reap(done chan bool) {
	filterables := allFilterables(r.errCh, r.infoCh)
	// TODO: consider slice of pointers
	var asgs []AutoScalingGroup
	for _, f := range filterables {
		switch t := f.(type) {
		case *Instance:
			reapInstance(t)
		case *AutoScalingGroup:
			reapAutoScalingGroup(t)
			asgs = append(asgs, *t)
		case *Snapshot:
			reapSnapshot(t)
		default:
			r.errCh <- fmt.Errorf("Reap found unhandleable type.")
		}
	}

	// TODO: this totally doesn't work because it happens too late
	// basically this doesn't do anything
	// identify instances in an ASG and delete them from Reapables
	instanceIDsInASGs := allASGInstanceIds(asgs)
	for region := range instanceIDsInASGs {
		for instanceID := range instanceIDsInASGs[region] {
			delete(Reapables[region], instanceID)
		}
	}

	done <- true
}

// makes a slice of all filterables by appending
// output of each filterable types aggregator function
func allFilterables(errc chan error, infoc chan string) []Filterable {
	var filterables []Filterable
	if Conf.Enabled.Instances {
		filterables = append(filterables, allInstances(errc, infoc)...)
	}
	if Conf.Enabled.AutoScalingGroups {
		filterables = append(filterables, allAutoScalingGroups(errc, infoc)...)
	}
	if Conf.Enabled.Snapshots {
		filterables = append(filterables, allSnapshots()...)
	}
	return filterables
}

// applies N functions to a filterable F
// returns true if all filters returned true, else returns false
func applyFilters(f Filterable, filters map[string]Filter) bool {
	// recover from potential panics caused by malformed filters
	defer func() {
		if r := recover(); r != nil {
			Log.Error(fmt.Sprintf("Recovered in applyFilters with panic: %s", r))
		}
	}()

	// defaults to a match
	matched := true

	// if any of the filters return false -> not a match
	for _, filter := range filters {
		if !f.Filter(filter) {
			matched = false
		}
	}

	// whitelist filter
	if f.Filter(Filter{"Tagged", []string{Conf.WhitelistTag}}) {
		// if the filterable matches this filter, then
		// it should be whitelisted, aka not matched
		matched = false
	}

	return matched
}

func reapSnapshot(s *Snapshot) {
	filters := Conf.Filters.Snapshot
	if applyFilters(s, filters) {
		Log.Debug(fmt.Sprintf("Snapshot %s matched %s.",
			s.ID,
			PrintFilters(filters)))
		// TODO
		// for _, e := range Events {
		// e.NewReapableSnapshotEvent(s)
		// }
	}
}

func reapInstance(i *Instance) {
	filters := Conf.Filters.Instance
	if applyFilters(i, filters) {
		ownerString := ""
		if owner := i.Owner(); owner != nil {
			ownerString = fmt.Sprintf("%s ", owner)
		}
		Log.Debug(fmt.Sprintf("Instance %s %sin region %s matched %s.",
			i.ID,
			ownerString,
			i.Region,
			PrintFilters(filters)))

		for _, e := range Events {
			go e.NewEvent("Reapable instance discovered", string(i.ReapableEventText().Bytes()), nil, nil)
			go e.NewStatistic("reaper.instances.reapable", 1, []string{fmt.Sprintf("id:%s", i.ID)})
		}

		// add to Reapables
		Reapables[i.Region][i.ID] = i

		// if the instance is owned, email the owner
		// sends different notification based on reaper state
		// currently there is a conifg option to enable these: Conf.Notifications.Extras
		if i.Owned() && Conf.Notifications.Extras {
			switch i.reaperState.State {
			case STATE_START:
				for _, e := range Events {
					go e.NewEvent("Reaper sent notification 1", fmt.Sprintf("Notification 1 sent to %s for instance %s.", i.Owner(), i.ID), nil, nil)
				}

			case STATE_NOTIFY1:
				for _, e := range Events {
					go e.NewEvent("Reaper sent notification 2", fmt.Sprintf("Notification 2 sent to %s for instance %s.", i.Owner(), i.ID), nil, nil)
				}

			case STATE_NOTIFY2:
				for _, e := range Events {
					go e.NewEvent("Reaper terminated instance", fmt.Sprintf("Instance owned by %s with id: %s was terminated.", i.Owner(), i.ID), nil, nil)
				}
			}
		}
	}
}

func reapAutoScalingGroup(a *AutoScalingGroup) {
	filters := Conf.Filters.ASG
	if applyFilters(a, filters) {
		Log.Debug(fmt.Sprintf("ASG %s matched %s.",
			a.ID,
			PrintFilters(filters)))

		for _, e := range Events {
			go e.NewEvent("Reapable ASG discovered", string(a.ReapableEventText().Bytes()), nil, nil)
		}
	}

	// add to Reapables
	Reapables[a.Region][a.ID] = a
}

func (r *Reaper) terminateUnowned(i *Instance) error {
	Log.Info("Terminate UNOWNED instance (%s) %s, owner tag: %s",
		i.ID, i.Name, i.Tag("Owner"))

	if Conf.DryRun {
		return nil
	}

	// TODO: use success here
	if _, err := i.Terminate(); err != nil {
		Log.Error(fmt.Sprintf("Terminate %s error: %s", i.ID, err.Error()))
		return err
	}

	return nil

}

// fetches a reapable matching region, id from
// the global slice of reapables
func getReapable(region, id string) (Reapable, error) {
	reapable, ok := Reapables[region][id]
	if !ok {
		Log.Error("Could not terminate resource with region: %s and id: %s.",
			region, id)
		return reapable, fmt.Errorf("No such resource.")
	}
	return reapable, nil
}

// Terminate by region, id, calls a Reapable's own Terminate method
func Terminate(region, id string) error {
	reapable, err := getReapable(region, id)
	if err != nil {
		return err
	}
	_, err = reapable.Terminate()
	if err != nil {
		Log.Error("Could not terminate resource with region: %s and id: %s. Error: %s",
			region, id, err.Error())
		return err
	}

	return nil
}

// ForceStop by region, id, calls a Reapable's own ForceStop method
func ForceStop(region, id string) error {
	reapable, err := getReapable(region, id)
	if err != nil {
		return err
	}
	_, err = reapable.ForceStop()
	if err != nil {
		Log.Error("Could not stop resource with region: %s and id: %s. Error: %s",
			region, id, err.Error())
		return err
	}

	return nil
}

// Stop by region, id, calls a Reapable's own Stop method
func Stop(region, id string) error {
	reapable, err := getReapable(region, id)
	if err != nil {
		return err
	}
	_, err = reapable.Stop()
	if err != nil {
		Log.Error("Could not stop resource with region: %s and id: %s. Error: %s",
			region, id, err.Error())
		return err
	}

	return nil
}

// allInstances describes every instance in the requested regions
// instances of Instance are created for each *ec2.Instance
// returned as Filterables
func allInstances(errc chan error, infoc chan string) []Filterable {

	regions := Conf.AWS.Regions
	var wg sync.WaitGroup
	in := make(chan *Instance)

	// fetch all info in parallel
	for _, region := range regions {
		wg.Add(1)

		go func(region string, errc chan error, infoc chan string) {
			defer wg.Done()
			api := ec2.New(&aws.Config{Region: region})

			/* //uncomment to enable a whole bunch of debug output
			api.Config.LogLevel = 1
			api.AddDebugHandlers()
			*/

			// repeat until we have everything
			var nextToken *string
			sum := 0

			for done := false; done != true; {
				input := &ec2.DescribeInstancesInput{
					NextToken: nextToken,
				}
				resp, err := api.DescribeInstances(input)
				if err != nil {
					// probably should do something here...
					errc <- fmt.Errorf("EC2 error in %s: %s", region, err.Error())
					return
				}

				for _, r := range resp.Reservations {
					for _, instance := range r.Instances {
						sum += 1
						in <- NewInstance(region, instance)
					}
				}

				if resp.NextToken != nil {
					infoc <- fmt.Sprintf("More results for DescribeInstances in %s", region)
					nextToken = resp.NextToken
				} else {
					done = true
				}
			}

			infoc <- fmt.Sprintf("Found %d total instances in %s", sum, region)
			for _, e := range Events {
				go func(c chan error, e events.EventReporter) {
					c <- e.NewStatistic("reaper.instances.total", float64(sum), []string{fmt.Sprintf("region:%s", region)})
				}(errc, e)
			}
		}(region, errc, infoc)
	}

	var list []Filterable

	// build up the list
	go func() {
		for i := range in {
			list = append(list, i)
		}
	}()

	// wait for all the fetches to finish publishing
	wg.Wait()
	close(in)

	infoc <- fmt.Sprintf("Found %d total instances.", len(list))
	return list
}
