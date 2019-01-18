package vspherecompute

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	biav1alpha1 "github.com/mylesgray/vspherecompute/pkg/apis/bia/v1alpha1"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// getEnvString returns string from environment variable.
func getEnvString(v string, def string) string {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	return r
}

// getEnvBool returns boolean from environment variable.
func getEnvBool(v string, def bool) bool {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	switch strings.ToLower(r[0:1]) {
	case "t", "y", "1":
		return true
	}

	return false
}

const (
	envURL      = "GOVC_URL"
	envUserName = "GOVC_USERNAME"
	envPassword = "GOVC_PASSWORD"
	envInsecure = "GOVC_INSECURE"
)

var urlDescription = fmt.Sprintf("ESX or vCenter URL [%s]", envURL)
var urlFlag = flag.String("url", getEnvString(envURL, "https://username:password@host"+vim25.Path), urlDescription)

var insecureDescription = fmt.Sprintf("Don't verify the server's certificate chain [%s]", envInsecure)
var insecureFlag = flag.Bool("insecure", getEnvBool(envInsecure, false), insecureDescription)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new VSphereCompute Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	flag.Parse()

	// Parse URL from string
	u, err := soap.ParseURL(*urlFlag)
	if err != nil {
		log.Println(err)
	}

	log.Println(u)

	return &ReconcileVSphereCompute{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vspherecompute-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VSphereCompute
	err = c.Watch(&source.Kind{Type: &biav1alpha1.VSphereCompute{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner VSphereCompute
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &biav1alpha1.VSphereCompute{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVSphereCompute{}

// ReconcileVSphereCompute reconciles a VSphereCompute object
type ReconcileVSphereCompute struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VSphereCompute object and makes changes based on the state read
// and what is in the VSphereCompute.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVSphereCompute) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling VSphereCompute %s/%s\n", request.Namespace, request.Name)

	// Fetch the VSphereCompute instance
	instance := &biav1alpha1.VSphereCompute{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Println("Deleting VM " + request.Name + "...")
			vminfojson, err := exec.Command("govc", "vm.info", "-json", request.Name).Output()
			if err != nil {
				log.Println(err)
				return reconcile.Result{}, err
			}
			var vminfo VM

			error := json.Unmarshal(vminfojson, &vminfo)
			if error != nil {
				log.Println(error)
				return reconcile.Result{}, error
			}

			error1 := exec.Command("govc", "vm.destroy", "-vm.uuid="+vminfo.VirtualMachines[0].Config.UUID).Run()
			if error1 != nil {
				log.Println(error1)
				return reconcile.Result{}, error1
			}
			log.Println("VM " + request.Name + " deleted")

			return reconcile.Result{}, err
		}
		// Error reading the object - requeue the request.
		log.Println("Failed to get VSphereCompute.")
		return reconcile.Result{}, err
	}

	name := instance.Spec.Name
	cpus := instance.Spec.VMCpu
	//template := instance.Spec.Template
	memory := instance.Spec.VMMemory
	//vmReservedResources := instance.Spec.VMReservedResources
	//replicas := instance.Spec.Replicas
	//affinity := instance.Spec.Affinity
	//vmAntiAffinity := instance.Spec.Affinity.VMAntiAffinity
	//topologyKey := instance.Spec.Affinity.VMAntiAffinity.TopologyKey
	//enforced := instance.Spec.Affinity.VMAntiAffinity.Enforced
	datastore := "vsanDatastore"

	// Check if items already exist, if not, create them
	vminfojson, err := exec.Command("govc", "vm.info", "-json", name).Output()
	if err != nil {
		log.Println(err)
	}

	var vminfo VM

	error := json.Unmarshal(vminfojson, &vminfo)
	if error != nil {
		log.Println(error)
	}

	if vminfo.VirtualMachines != nil {
		log.Println("VM " + name + " already exists, doing nothing.")
	} else {
		log.Println("VM " + name + " doesn't exist, creating...")

		err := exec.Command("govc", "vm.create", "-json=true", "-c="+strconv.Itoa(cpus), "-m="+strconv.Itoa(memory), "-ds="+datastore, name).Run()
		if err != nil {
			log.Println(err)
			return reconcile.Result{}, err
		}
		log.Println("VM " + name + " created")

		vminfojson, err := exec.Command("govc", "vm.info", "-json", name).Output()
		if err != nil {
			log.Println(err)
			return reconcile.Result{}, err
		}

		error := json.Unmarshal(vminfojson, &vminfo)
		if error != nil {
			log.Println(error)
		}
	}

	instance.Status.CPU = vminfo.VirtualMachines[0].Config.Hardware.NumCPU
	instance.Status.Memory = vminfo.VirtualMachines[0].Config.Hardware.MemoryMB
	instance.Status.Status = vminfo.VirtualMachines[0].Summary.Runtime.PowerState
	instance.Status.IP = vminfo.VirtualMachines[0].Guest.IPAddress
	instance.Status.Host = vminfo.VirtualMachines[0].Runtime.Host.Value
	instance.Status.VMName = vminfo.VirtualMachines[0].Name

	r.client.Status().Update(context.TODO(), instance)

	return reconcile.Result{}, nil
}

// VM type for JSON reply from govc
type VM struct {
	VirtualMachines []struct {
		Self struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"Self"`
		Value          interface{} `json:"Value"`
		AvailableField interface{} `json:"AvailableField"`
		Parent         struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"Parent"`
		CustomValue        interface{} `json:"CustomValue"`
		OverallStatus      string      `json:"OverallStatus"`
		ConfigStatus       string      `json:"ConfigStatus"`
		ConfigIssue        interface{} `json:"ConfigIssue"`
		EffectiveRole      []int       `json:"EffectiveRole"`
		Permission         interface{} `json:"Permission"`
		Name               string      `json:"Name"`
		DisabledMethod     []string    `json:"DisabledMethod"`
		RecentTask         interface{} `json:"RecentTask"`
		DeclaredAlarmState []struct {
			Key    string `json:"Key"`
			Entity struct {
				Type  string `json:"Type"`
				Value string `json:"Value"`
			} `json:"Entity"`
			Alarm struct {
				Type  string `json:"Type"`
				Value string `json:"Value"`
			} `json:"Alarm"`
			OverallStatus      string      `json:"OverallStatus"`
			Time               time.Time   `json:"Time"`
			Acknowledged       bool        `json:"Acknowledged"`
			AcknowledgedByUser string      `json:"AcknowledgedByUser"`
			AcknowledgedTime   interface{} `json:"AcknowledgedTime"`
			EventKey           int         `json:"EventKey"`
		} `json:"DeclaredAlarmState"`
		TriggeredAlarmState interface{} `json:"TriggeredAlarmState"`
		AlarmActionsEnabled bool        `json:"AlarmActionsEnabled"`
		Tag                 interface{} `json:"Tag"`
		Capability          struct {
			SnapshotOperationsSupported            bool `json:"SnapshotOperationsSupported"`
			MultipleSnapshotsSupported             bool `json:"MultipleSnapshotsSupported"`
			SnapshotConfigSupported                bool `json:"SnapshotConfigSupported"`
			PoweredOffSnapshotsSupported           bool `json:"PoweredOffSnapshotsSupported"`
			MemorySnapshotsSupported               bool `json:"MemorySnapshotsSupported"`
			RevertToSnapshotSupported              bool `json:"RevertToSnapshotSupported"`
			QuiescedSnapshotsSupported             bool `json:"QuiescedSnapshotsSupported"`
			DisableSnapshotsSupported              bool `json:"DisableSnapshotsSupported"`
			LockSnapshotsSupported                 bool `json:"LockSnapshotsSupported"`
			ConsolePreferencesSupported            bool `json:"ConsolePreferencesSupported"`
			CPUFeatureMaskSupported                bool `json:"CpuFeatureMaskSupported"`
			S1AcpiManagementSupported              bool `json:"S1AcpiManagementSupported"`
			SettingScreenResolutionSupported       bool `json:"SettingScreenResolutionSupported"`
			ToolsAutoUpdateSupported               bool `json:"ToolsAutoUpdateSupported"`
			VMNpivWwnSupported                     bool `json:"VmNpivWwnSupported"`
			NpivWwnOnNonRdmVMSupported             bool `json:"NpivWwnOnNonRdmVmSupported"`
			VMNpivWwnDisableSupported              bool `json:"VmNpivWwnDisableSupported"`
			VMNpivWwnUpdateSupported               bool `json:"VmNpivWwnUpdateSupported"`
			SwapPlacementSupported                 bool `json:"SwapPlacementSupported"`
			ToolsSyncTimeSupported                 bool `json:"ToolsSyncTimeSupported"`
			VirtualMmuUsageSupported               bool `json:"VirtualMmuUsageSupported"`
			DiskSharesSupported                    bool `json:"DiskSharesSupported"`
			BootOptionsSupported                   bool `json:"BootOptionsSupported"`
			BootRetryOptionsSupported              bool `json:"BootRetryOptionsSupported"`
			SettingVideoRAMSizeSupported           bool `json:"SettingVideoRamSizeSupported"`
			SettingDisplayTopologySupported        bool `json:"SettingDisplayTopologySupported"`
			RecordReplaySupported                  bool `json:"RecordReplaySupported"`
			ChangeTrackingSupported                bool `json:"ChangeTrackingSupported"`
			MultipleCoresPerSocketSupported        bool `json:"MultipleCoresPerSocketSupported"`
			HostBasedReplicationSupported          bool `json:"HostBasedReplicationSupported"`
			GuestAutoLockSupported                 bool `json:"GuestAutoLockSupported"`
			MemoryReservationLockSupported         bool `json:"MemoryReservationLockSupported"`
			FeatureRequirementSupported            bool `json:"FeatureRequirementSupported"`
			PoweredOnMonitorTypeChangeSupported    bool `json:"PoweredOnMonitorTypeChangeSupported"`
			SeSparseDiskSupported                  bool `json:"SeSparseDiskSupported"`
			NestedHVSupported                      bool `json:"NestedHVSupported"`
			VPMCSupported                          bool `json:"VPMCSupported"`
			SecureBootSupported                    bool `json:"SecureBootSupported"`
			PerVMEvcSupported                      bool `json:"PerVmEvcSupported"`
			VirtualMmuUsageIgnored                 bool `json:"VirtualMmuUsageIgnored"`
			VirtualExecUsageIgnored                bool `json:"VirtualExecUsageIgnored"`
			DiskOnlySnapshotOnSuspendedVMSupported bool `json:"DiskOnlySnapshotOnSuspendedVMSupported"`
		} `json:"Capability"`
		Config struct {
			ChangeVersion         time.Time   `json:"ChangeVersion"`
			Modified              time.Time   `json:"Modified"`
			Name                  string      `json:"Name"`
			GuestFullName         string      `json:"GuestFullName"`
			Version               string      `json:"Version"`
			UUID                  string      `json:"Uuid"`
			CreateDate            time.Time   `json:"CreateDate"`
			InstanceUUID          string      `json:"InstanceUuid"`
			NpivNodeWorldWideName interface{} `json:"NpivNodeWorldWideName"`
			NpivPortWorldWideName interface{} `json:"NpivPortWorldWideName"`
			NpivWorldWideNameType string      `json:"NpivWorldWideNameType"`
			NpivDesiredNodeWwns   int         `json:"NpivDesiredNodeWwns"`
			NpivDesiredPortWwns   int         `json:"NpivDesiredPortWwns"`
			NpivTemporaryDisabled bool        `json:"NpivTemporaryDisabled"`
			NpivOnNonRdmDisks     interface{} `json:"NpivOnNonRdmDisks"`
			LocationID            string      `json:"LocationId"`
			Template              bool        `json:"Template"`
			GuestID               string      `json:"GuestId"`
			AlternateGuestName    string      `json:"AlternateGuestName"`
			Annotation            string      `json:"Annotation"`
			Files                 struct {
				VMPathName          string `json:"VmPathName"`
				SnapshotDirectory   string `json:"SnapshotDirectory"`
				SuspendDirectory    string `json:"SuspendDirectory"`
				LogDirectory        string `json:"LogDirectory"`
				FtMetadataDirectory string `json:"FtMetadataDirectory"`
			} `json:"Files"`
			Tools struct {
				ToolsVersion         int         `json:"ToolsVersion"`
				ToolsInstallType     string      `json:"ToolsInstallType"`
				AfterPowerOn         bool        `json:"AfterPowerOn"`
				AfterResume          bool        `json:"AfterResume"`
				BeforeGuestStandby   bool        `json:"BeforeGuestStandby"`
				BeforeGuestShutdown  bool        `json:"BeforeGuestShutdown"`
				BeforeGuestReboot    interface{} `json:"BeforeGuestReboot"`
				ToolsUpgradePolicy   string      `json:"ToolsUpgradePolicy"`
				PendingCustomization string      `json:"PendingCustomization"`
				CustomizationKeyID   interface{} `json:"CustomizationKeyId"`
				SyncTimeWithHost     bool        `json:"SyncTimeWithHost"`
				LastInstallInfo      struct {
					Counter int         `json:"Counter"`
					Fault   interface{} `json:"Fault"`
				} `json:"LastInstallInfo"`
			} `json:"Tools"`
			Flags struct {
				DisableAcceleration      bool   `json:"DisableAcceleration"`
				EnableLogging            bool   `json:"EnableLogging"`
				UseToe                   bool   `json:"UseToe"`
				RunWithDebugInfo         bool   `json:"RunWithDebugInfo"`
				MonitorType              string `json:"MonitorType"`
				HtSharing                string `json:"HtSharing"`
				SnapshotDisabled         bool   `json:"SnapshotDisabled"`
				SnapshotLocked           bool   `json:"SnapshotLocked"`
				DiskUUIDEnabled          bool   `json:"DiskUuidEnabled"`
				VirtualMmuUsage          string `json:"VirtualMmuUsage"`
				VirtualExecUsage         string `json:"VirtualExecUsage"`
				SnapshotPowerOffBehavior string `json:"SnapshotPowerOffBehavior"`
				RecordReplayEnabled      bool   `json:"RecordReplayEnabled"`
				FaultToleranceType       string `json:"FaultToleranceType"`
				CbrcCacheEnabled         bool   `json:"CbrcCacheEnabled"`
				VvtdEnabled              bool   `json:"VvtdEnabled"`
				VbsEnabled               bool   `json:"VbsEnabled"`
			} `json:"Flags"`
			ConsolePreferences interface{} `json:"ConsolePreferences"`
			DefaultPowerOps    struct {
				PowerOffType        string `json:"PowerOffType"`
				SuspendType         string `json:"SuspendType"`
				ResetType           string `json:"ResetType"`
				DefaultPowerOffType string `json:"DefaultPowerOffType"`
				DefaultSuspendType  string `json:"DefaultSuspendType"`
				DefaultResetType    string `json:"DefaultResetType"`
				StandbyAction       string `json:"StandbyAction"`
			} `json:"DefaultPowerOps"`
			Hardware struct {
				NumCPU              int  `json:"NumCPU"`
				NumCoresPerSocket   int  `json:"NumCoresPerSocket"`
				MemoryMB            int  `json:"MemoryMB"`
				VirtualICH7MPresent bool `json:"VirtualICH7MPresent"`
				VirtualSMCPresent   bool `json:"VirtualSMCPresent"`
				Device              []struct {
					Key        int `json:"Key"`
					DeviceInfo struct {
						Label   string `json:"Label"`
						Summary string `json:"Summary"`
					} `json:"DeviceInfo"`
					Backing                        interface{} `json:"Backing"`
					Connectable                    interface{} `json:"Connectable"`
					SlotInfo                       interface{} `json:"SlotInfo"`
					ControllerKey                  int         `json:"ControllerKey"`
					UnitNumber                     interface{} `json:"UnitNumber"`
					BusNumber                      int         `json:"BusNumber,omitempty"`
					Device                         interface{} `json:"Device,omitempty"`
					VideoRAMSizeInKB               int         `json:"VideoRamSizeInKB,omitempty"`
					NumDisplays                    int         `json:"NumDisplays,omitempty"`
					UseAutoDetect                  bool        `json:"UseAutoDetect,omitempty"`
					Enable3DSupport                bool        `json:"Enable3DSupport,omitempty"`
					Use3DRenderer                  string      `json:"Use3dRenderer,omitempty"`
					GraphicsMemorySizeInKB         int         `json:"GraphicsMemorySizeInKB,omitempty"`
					ID                             int         `json:"Id,omitempty"`
					AllowUnrestrictedCommunication bool        `json:"AllowUnrestrictedCommunication,omitempty"`
					FilterEnable                   bool        `json:"FilterEnable,omitempty"`
					FilterInfo                     interface{} `json:"FilterInfo,omitempty"`
					HotAddRemove                   bool        `json:"HotAddRemove,omitempty"`
					SharedBus                      string      `json:"SharedBus,omitempty"`
					ScsiCtlrUnitNumber             int         `json:"ScsiCtlrUnitNumber,omitempty"`
					AddressType                    string      `json:"AddressType,omitempty"`
					MacAddress                     string      `json:"MacAddress,omitempty"`
					WakeOnLanEnabled               bool        `json:"WakeOnLanEnabled,omitempty"`
					ResourceAllocation             struct {
						Reservation int `json:"Reservation"`
						Share       struct {
							Shares int    `json:"Shares"`
							Level  string `json:"Level"`
						} `json:"Share"`
						Limit int `json:"Limit"`
					} `json:"ResourceAllocation,omitempty"`
					ExternalID              string `json:"ExternalId,omitempty"`
					UptCompatibilityEnabled bool   `json:"UptCompatibilityEnabled,omitempty"`
				} `json:"Device"`
			} `json:"Hardware"`
			CPUAllocation struct {
				Reservation           int  `json:"Reservation"`
				ExpandableReservation bool `json:"ExpandableReservation"`
				Limit                 int  `json:"Limit"`
				Shares                struct {
					Shares int    `json:"Shares"`
					Level  string `json:"Level"`
				} `json:"Shares"`
				OverheadLimit interface{} `json:"OverheadLimit"`
			} `json:"CpuAllocation"`
			MemoryAllocation struct {
				Reservation           int  `json:"Reservation"`
				ExpandableReservation bool `json:"ExpandableReservation"`
				Limit                 int  `json:"Limit"`
				Shares                struct {
					Shares int    `json:"Shares"`
					Level  string `json:"Level"`
				} `json:"Shares"`
				OverheadLimit interface{} `json:"OverheadLimit"`
			} `json:"MemoryAllocation"`
			LatencySensitivity struct {
				Level       string `json:"Level"`
				Sensitivity int    `json:"Sensitivity"`
			} `json:"LatencySensitivity"`
			MemoryHotAddEnabled        bool        `json:"MemoryHotAddEnabled"`
			CPUHotAddEnabled           bool        `json:"CpuHotAddEnabled"`
			CPUHotRemoveEnabled        bool        `json:"CpuHotRemoveEnabled"`
			HotPlugMemoryLimit         int         `json:"HotPlugMemoryLimit"`
			HotPlugMemoryIncrementSize int         `json:"HotPlugMemoryIncrementSize"`
			CPUAffinity                interface{} `json:"CpuAffinity"`
			MemoryAffinity             interface{} `json:"MemoryAffinity"`
			NetworkShaper              interface{} `json:"NetworkShaper"`
			ExtraConfig                []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"ExtraConfig"`
			CPUFeatureMask interface{} `json:"CpuFeatureMask"`
			DatastoreURL   []struct {
				Name string `json:"Name"`
				URL  string `json:"Url"`
			} `json:"DatastoreUrl"`
			SwapPlacement string `json:"SwapPlacement"`
			BootOptions   struct {
				BootDelay            int         `json:"BootDelay"`
				EnterBIOSSetup       bool        `json:"EnterBIOSSetup"`
				EfiSecureBootEnabled bool        `json:"EfiSecureBootEnabled"`
				BootRetryEnabled     bool        `json:"BootRetryEnabled"`
				BootRetryDelay       int         `json:"BootRetryDelay"`
				BootOrder            interface{} `json:"BootOrder"`
				NetworkBootProtocol  string      `json:"NetworkBootProtocol"`
			} `json:"BootOptions"`
			FtInfo                       interface{} `json:"FtInfo"`
			RepConfig                    interface{} `json:"RepConfig"`
			VAppConfig                   interface{} `json:"VAppConfig"`
			VAssertsEnabled              bool        `json:"VAssertsEnabled"`
			ChangeTrackingEnabled        bool        `json:"ChangeTrackingEnabled"`
			Firmware                     string      `json:"Firmware"`
			MaxMksConnections            int         `json:"MaxMksConnections"`
			GuestAutoLockEnabled         bool        `json:"GuestAutoLockEnabled"`
			ManagedBy                    interface{} `json:"ManagedBy"`
			MemoryReservationLockedToMax bool        `json:"MemoryReservationLockedToMax"`
			InitialOverhead              struct {
				InitialMemoryReservation int `json:"InitialMemoryReservation"`
				InitialSwapReservation   int `json:"InitialSwapReservation"`
			} `json:"InitialOverhead"`
			NestedHVEnabled              bool `json:"NestedHVEnabled"`
			VPMCEnabled                  bool `json:"VPMCEnabled"`
			ScheduledHardwareUpgradeInfo struct {
				UpgradePolicy                  string      `json:"UpgradePolicy"`
				VersionKey                     string      `json:"VersionKey"`
				ScheduledHardwareUpgradeStatus string      `json:"ScheduledHardwareUpgradeStatus"`
				Fault                          interface{} `json:"Fault"`
			} `json:"ScheduledHardwareUpgradeInfo"`
			ForkConfigInfo          interface{} `json:"ForkConfigInfo"`
			VFlashCacheReservation  int         `json:"VFlashCacheReservation"`
			VmxConfigChecksum       string      `json:"VmxConfigChecksum"`
			MessageBusTunnelEnabled bool        `json:"MessageBusTunnelEnabled"`
			VMStorageObjectID       string      `json:"VmStorageObjectId"`
			SwapStorageObjectID     string      `json:"SwapStorageObjectId"`
			KeyID                   interface{} `json:"KeyId"`
			GuestIntegrityInfo      struct {
				Enabled bool `json:"Enabled"`
			} `json:"GuestIntegrityInfo"`
			MigrateEncryption string `json:"MigrateEncryption"`
		} `json:"Config"`
		Layout struct {
			ConfigFile []string    `json:"ConfigFile"`
			LogFile    interface{} `json:"LogFile"`
			Disk       interface{} `json:"Disk"`
			Snapshot   interface{} `json:"Snapshot"`
			SwapFile   string      `json:"SwapFile"`
		} `json:"Layout"`
		LayoutEx struct {
			File []struct {
				Key             int    `json:"Key"`
				Name            string `json:"Name"`
				Type            string `json:"Type"`
				Size            int    `json:"Size"`
				UniqueSize      int    `json:"UniqueSize"`
				BackingObjectID string `json:"BackingObjectId"`
				Accessible      bool   `json:"Accessible"`
			} `json:"File"`
			Disk      interface{} `json:"Disk"`
			Snapshot  interface{} `json:"Snapshot"`
			Timestamp time.Time   `json:"Timestamp"`
		} `json:"LayoutEx"`
		Storage struct {
			PerDatastoreUsage []struct {
				Datastore struct {
					Type  string `json:"Type"`
					Value string `json:"Value"`
				} `json:"Datastore"`
				Committed   int64 `json:"Committed"`
				Uncommitted int   `json:"Uncommitted"`
				Unshared    int   `json:"Unshared"`
			} `json:"PerDatastoreUsage"`
			Timestamp time.Time `json:"Timestamp"`
		} `json:"Storage"`
		EnvironmentBrowser struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"EnvironmentBrowser"`
		ResourcePool struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"ResourcePool"`
		ParentVApp     interface{} `json:"ParentVApp"`
		ResourceConfig struct {
			Entity struct {
				Type  string `json:"Type"`
				Value string `json:"Value"`
			} `json:"Entity"`
			ChangeVersion string      `json:"ChangeVersion"`
			LastModified  interface{} `json:"LastModified"`
			CPUAllocation struct {
				Reservation           int  `json:"Reservation"`
				ExpandableReservation bool `json:"ExpandableReservation"`
				Limit                 int  `json:"Limit"`
				Shares                struct {
					Shares int    `json:"Shares"`
					Level  string `json:"Level"`
				} `json:"Shares"`
				OverheadLimit interface{} `json:"OverheadLimit"`
			} `json:"CpuAllocation"`
			MemoryAllocation struct {
				Reservation           int  `json:"Reservation"`
				ExpandableReservation bool `json:"ExpandableReservation"`
				Limit                 int  `json:"Limit"`
				Shares                struct {
					Shares int    `json:"Shares"`
					Level  string `json:"Level"`
				} `json:"Shares"`
				OverheadLimit interface{} `json:"OverheadLimit"`
			} `json:"MemoryAllocation"`
		} `json:"ResourceConfig"`
		Runtime struct {
			Device []struct {
				RuntimeState struct {
					VMDirectPathGen2Active                 bool        `json:"VmDirectPathGen2Active"`
					VMDirectPathGen2InactiveReasonVM       interface{} `json:"VmDirectPathGen2InactiveReasonVm"`
					VMDirectPathGen2InactiveReasonOther    []string    `json:"VmDirectPathGen2InactiveReasonOther"`
					VMDirectPathGen2InactiveReasonExtended string      `json:"VmDirectPathGen2InactiveReasonExtended"`
					ReservationStatus                      string      `json:"ReservationStatus"`
					AttachmentStatus                       string      `json:"AttachmentStatus"`
					FeatureRequirement                     interface{} `json:"FeatureRequirement"`
				} `json:"RuntimeState"`
				Key int `json:"Key"`
			} `json:"Device"`
			Host struct {
				Type  string `json:"Type"`
				Value string `json:"Value"`
			} `json:"Host"`
			ConnectionState           string      `json:"ConnectionState"`
			PowerState                string      `json:"PowerState"`
			FaultToleranceState       string      `json:"FaultToleranceState"`
			DasVMProtection           interface{} `json:"DasVmProtection"`
			ToolsInstallerMounted     bool        `json:"ToolsInstallerMounted"`
			SuspendTime               interface{} `json:"SuspendTime"`
			BootTime                  time.Time   `json:"BootTime"`
			SuspendInterval           int         `json:"SuspendInterval"`
			Question                  interface{} `json:"Question"`
			MemoryOverhead            int         `json:"MemoryOverhead"`
			MaxCPUUsage               int         `json:"MaxCpuUsage"`
			MaxMemoryUsage            int         `json:"MaxMemoryUsage"`
			NumMksConnections         int         `json:"NumMksConnections"`
			RecordReplayState         string      `json:"RecordReplayState"`
			CleanPowerOff             interface{} `json:"CleanPowerOff"`
			NeedSecondaryReason       string      `json:"NeedSecondaryReason"`
			OnlineStandby             bool        `json:"OnlineStandby"`
			MinRequiredEVCModeKey     string      `json:"MinRequiredEVCModeKey"`
			ConsolidationNeeded       bool        `json:"ConsolidationNeeded"`
			OfflineFeatureRequirement interface{} `json:"OfflineFeatureRequirement"`
			FeatureRequirement        []struct {
				Key         string `json:"Key"`
				FeatureName string `json:"FeatureName"`
				Value       string `json:"Value"`
			} `json:"FeatureRequirement"`
			FeatureMask           interface{} `json:"FeatureMask"`
			VFlashCacheAllocation int         `json:"VFlashCacheAllocation"`
			Paused                bool        `json:"Paused"`
			SnapshotInBackground  bool        `json:"SnapshotInBackground"`
			QuiescedForkParent    interface{} `json:"QuiescedForkParent"`
			InstantCloneFrozen    bool        `json:"InstantCloneFrozen"`
			CryptoState           string      `json:"CryptoState"`
		} `json:"Runtime"`
		Guest struct {
			ToolsStatus         string      `json:"ToolsStatus"`
			ToolsVersionStatus  string      `json:"ToolsVersionStatus"`
			ToolsVersionStatus2 string      `json:"ToolsVersionStatus2"`
			ToolsRunningStatus  string      `json:"ToolsRunningStatus"`
			ToolsVersion        string      `json:"ToolsVersion"`
			ToolsInstallType    string      `json:"ToolsInstallType"`
			GuestID             string      `json:"GuestId"`
			GuestFamily         string      `json:"GuestFamily"`
			GuestFullName       string      `json:"GuestFullName"`
			HostName            string      `json:"HostName"`
			IPAddress           string      `json:"IpAddress"`
			Net                 interface{} `json:"Net"`
			IPStack             interface{} `json:"IpStack"`
			Disk                interface{} `json:"Disk"`
			Screen              struct {
				Width  int `json:"Width"`
				Height int `json:"Height"`
			} `json:"Screen"`
			GuestState                      string      `json:"GuestState"`
			AppHeartbeatStatus              string      `json:"AppHeartbeatStatus"`
			GuestKernelCrashed              bool        `json:"GuestKernelCrashed"`
			AppState                        string      `json:"AppState"`
			GuestOperationsReady            bool        `json:"GuestOperationsReady"`
			InteractiveGuestOperationsReady bool        `json:"InteractiveGuestOperationsReady"`
			GuestStateChangeSupported       bool        `json:"GuestStateChangeSupported"`
			GenerationInfo                  interface{} `json:"GenerationInfo"`
		} `json:"Guest"`
		Summary struct {
			VM struct {
				Type  string `json:"Type"`
				Value string `json:"Value"`
			} `json:"Vm"`
			Runtime struct {
				Device []struct {
					RuntimeState struct {
						VMDirectPathGen2Active                 bool        `json:"VmDirectPathGen2Active"`
						VMDirectPathGen2InactiveReasonVM       interface{} `json:"VmDirectPathGen2InactiveReasonVm"`
						VMDirectPathGen2InactiveReasonOther    []string    `json:"VmDirectPathGen2InactiveReasonOther"`
						VMDirectPathGen2InactiveReasonExtended string      `json:"VmDirectPathGen2InactiveReasonExtended"`
						ReservationStatus                      string      `json:"ReservationStatus"`
						AttachmentStatus                       string      `json:"AttachmentStatus"`
						FeatureRequirement                     interface{} `json:"FeatureRequirement"`
					} `json:"RuntimeState"`
					Key int `json:"Key"`
				} `json:"Device"`
				Host struct {
					Type  string `json:"Type"`
					Value string `json:"Value"`
				} `json:"Host"`
				ConnectionState           string      `json:"ConnectionState"`
				PowerState                string      `json:"PowerState"`
				FaultToleranceState       string      `json:"FaultToleranceState"`
				DasVMProtection           interface{} `json:"DasVmProtection"`
				ToolsInstallerMounted     bool        `json:"ToolsInstallerMounted"`
				SuspendTime               interface{} `json:"SuspendTime"`
				BootTime                  time.Time   `json:"BootTime"`
				SuspendInterval           int         `json:"SuspendInterval"`
				Question                  interface{} `json:"Question"`
				MemoryOverhead            int         `json:"MemoryOverhead"`
				MaxCPUUsage               int         `json:"MaxCpuUsage"`
				MaxMemoryUsage            int         `json:"MaxMemoryUsage"`
				NumMksConnections         int         `json:"NumMksConnections"`
				RecordReplayState         string      `json:"RecordReplayState"`
				CleanPowerOff             interface{} `json:"CleanPowerOff"`
				NeedSecondaryReason       string      `json:"NeedSecondaryReason"`
				OnlineStandby             bool        `json:"OnlineStandby"`
				MinRequiredEVCModeKey     string      `json:"MinRequiredEVCModeKey"`
				ConsolidationNeeded       bool        `json:"ConsolidationNeeded"`
				OfflineFeatureRequirement interface{} `json:"OfflineFeatureRequirement"`
				FeatureRequirement        []struct {
					Key         string `json:"Key"`
					FeatureName string `json:"FeatureName"`
					Value       string `json:"Value"`
				} `json:"FeatureRequirement"`
				FeatureMask           interface{} `json:"FeatureMask"`
				VFlashCacheAllocation int         `json:"VFlashCacheAllocation"`
				Paused                bool        `json:"Paused"`
				SnapshotInBackground  bool        `json:"SnapshotInBackground"`
				QuiescedForkParent    interface{} `json:"QuiescedForkParent"`
				InstantCloneFrozen    bool        `json:"InstantCloneFrozen"`
				CryptoState           string      `json:"CryptoState"`
			} `json:"Runtime"`
			Guest struct {
				GuestID             string `json:"GuestId"`
				GuestFullName       string `json:"GuestFullName"`
				ToolsStatus         string `json:"ToolsStatus"`
				ToolsVersionStatus  string `json:"ToolsVersionStatus"`
				ToolsVersionStatus2 string `json:"ToolsVersionStatus2"`
				ToolsRunningStatus  string `json:"ToolsRunningStatus"`
				HostName            string `json:"HostName"`
				IPAddress           string `json:"IpAddress"`
			} `json:"Guest"`
			Config struct {
				Name                string      `json:"Name"`
				Template            bool        `json:"Template"`
				VMPathName          string      `json:"VmPathName"`
				MemorySizeMB        int         `json:"MemorySizeMB"`
				CPUReservation      int         `json:"CpuReservation"`
				MemoryReservation   int         `json:"MemoryReservation"`
				NumCPU              int         `json:"NumCpu"`
				NumEthernetCards    int         `json:"NumEthernetCards"`
				NumVirtualDisks     int         `json:"NumVirtualDisks"`
				UUID                string      `json:"Uuid"`
				InstanceUUID        string      `json:"InstanceUuid"`
				GuestID             string      `json:"GuestId"`
				GuestFullName       string      `json:"GuestFullName"`
				Annotation          string      `json:"Annotation"`
				Product             interface{} `json:"Product"`
				InstallBootRequired bool        `json:"InstallBootRequired"`
				FtInfo              interface{} `json:"FtInfo"`
				ManagedBy           interface{} `json:"ManagedBy"`
				TpmPresent          bool        `json:"TpmPresent"`
				NumVmiopBackings    int         `json:"NumVmiopBackings"`
			} `json:"Config"`
			Storage struct {
				Committed   int64     `json:"Committed"`
				Uncommitted int       `json:"Uncommitted"`
				Unshared    int       `json:"Unshared"`
				Timestamp   time.Time `json:"Timestamp"`
			} `json:"Storage"`
			QuickStats struct {
				OverallCPUUsage              int    `json:"OverallCpuUsage"`
				OverallCPUDemand             int    `json:"OverallCpuDemand"`
				GuestMemoryUsage             int    `json:"GuestMemoryUsage"`
				HostMemoryUsage              int    `json:"HostMemoryUsage"`
				GuestHeartbeatStatus         string `json:"GuestHeartbeatStatus"`
				DistributedCPUEntitlement    int    `json:"DistributedCpuEntitlement"`
				DistributedMemoryEntitlement int    `json:"DistributedMemoryEntitlement"`
				StaticCPUEntitlement         int    `json:"StaticCpuEntitlement"`
				StaticMemoryEntitlement      int    `json:"StaticMemoryEntitlement"`
				PrivateMemory                int    `json:"PrivateMemory"`
				SharedMemory                 int    `json:"SharedMemory"`
				SwappedMemory                int    `json:"SwappedMemory"`
				BalloonedMemory              int    `json:"BalloonedMemory"`
				ConsumedOverheadMemory       int    `json:"ConsumedOverheadMemory"`
				FtLogBandwidth               int    `json:"FtLogBandwidth"`
				FtSecondaryLatency           int    `json:"FtSecondaryLatency"`
				FtLatencyStatus              string `json:"FtLatencyStatus"`
				CompressedMemory             int    `json:"CompressedMemory"`
				UptimeSeconds                int    `json:"UptimeSeconds"`
				SsdSwappedMemory             int    `json:"SsdSwappedMemory"`
			} `json:"QuickStats"`
			OverallStatus string      `json:"OverallStatus"`
			CustomValue   interface{} `json:"CustomValue"`
		} `json:"Summary"`
		Datastore []struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"Datastore"`
		Network []struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"Network"`
		Snapshot             interface{} `json:"Snapshot"`
		RootSnapshot         interface{} `json:"RootSnapshot"`
		GuestHeartbeatStatus string      `json:"GuestHeartbeatStatus"`
	} `json:"VirtualMachines"`
}
