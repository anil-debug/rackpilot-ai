package redfish

import (
	"encoding/json"
	"net/http"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"
)

// Router serves Redfish resources from a snapshot provider.
type Router struct {
	h      *Handler
	lookup func() domain.Snapshot
}

func NewRouter(publicBase string, lookup func() domain.Snapshot) *Router {
	return &Router{h: NewHandler(publicBase), lookup: lookup}
}

func (r *Router) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /redfish/v1/", r.serviceRoot)
	mux.HandleFunc("GET /redfish/v1", r.serviceRoot)
	mux.HandleFunc("GET /redfish/v1/Systems", r.systemCollection)
	mux.HandleFunc("GET /redfish/v1/Systems/{id}", r.system)
	mux.HandleFunc("GET /redfish/v1/Chassis", r.chassisCollection)
	mux.HandleFunc("GET /redfish/v1/Chassis/{id}", r.chassis)
	mux.HandleFunc("GET /redfish/v1/Chassis/{id}/Thermal", r.thermal)
	mux.HandleFunc("GET /redfish/v1/Chassis/{id}/Power", r.power)
	mux.HandleFunc("GET /redfish/v1/Managers", r.managerCollection)
	mux.HandleFunc("GET /redfish/v1/Managers/{id}", r.manager)
	mux.HandleFunc("GET /redfish/v1/UpdateService/FirmwareInventory", r.firmwareInventory)
	mux.HandleFunc("GET /redfish/v1/Systems/{id}/LogServices", r.logServices)
	mux.HandleFunc("GET /redfish/v1/Systems/{id}/LogServices/EventLog/Entries", r.eventEntries)
}

func (r *Router) serviceRoot(w http.ResponseWriter, _ *http.Request) {
	s := r.lookup()
	body := map[string]any{
		"@odata.context": r.h.path("/redfish/v1/$metadata#ServiceRoot.ServiceRoot"),
		"@odata.id":      r.h.path("/redfish/v1/"),
		"@odata.type":    schemaHost + "/ServiceRoot.v1_15_0.ServiceRoot",
		"Id":             "RootService",
		"Name":           "Root Service",
		"RedfishVersion": "1.15.0",
		"UUID":           s.SystemID + "-synthetic",
		"Systems": map[string]any{
			"@odata.id": r.h.path("/redfish/v1/Systems"),
		},
		"Chassis": map[string]any{
			"@odata.id": r.h.path("/redfish/v1/Chassis"),
		},
		"Managers": map[string]any{
			"@odata.id": r.h.path("/redfish/v1/Managers"),
		},
		"Oem": map[string]any{
			"RackPilot": map[string]any{
				"SyntheticTelemetry": true,
				"Disclaimer":         "Synthetic BMC simulator — not a certified Redfish implementation.",
			},
		},
	}
	writeJSON(w, body)
}

func (r *Router) systemCollection(w http.ResponseWriter, _ *http.Request) {
	s := r.lookup()
	writeJSON(w, map[string]any{
		"@odata.context": r.h.path("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		"@odata.id":      r.h.path("/redfish/v1/Systems"),
		"@odata.type":    schemaHost + "/ComputerSystemCollection.ComputerSystemCollection",
		"Name":           "Computer System Collection",
		"Members": []map[string]any{
			{"@odata.id": r.h.path("/redfish/v1/Systems/" + s.SystemID)},
		},
		"Members@odata.count": 1,
	})
}

func (r *Router) system(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.SystemID {
		notFound(w)
		return
	}
	health := domain.RedfishHealth(s.SystemHealth)
	writeJSON(w, map[string]any{
		"@odata.context": r.h.path("/redfish/v1/$metadata#ComputerSystem.ComputerSystem"),
		"@odata.id":      r.h.path("/redfish/v1/Systems/" + id),
		"@odata.type":    schemaHost + "/ComputerSystem.v1_16_0.ComputerSystem",
		"Id":             id,
		"Name":           "Synthetic System " + id,
		"Manufacturer":   s.Manufacturer + " (simulated)",
		"Model":          s.Model,
		"PowerState":     s.PowerState,
		"SerialNumber":   "SYNTH-" + id,
		"Status":         statusBlock(health),
		"ProcessorSummary": map[string]any{
			"Count":  len(s.CPUs),
			"Status": statusBlock(health),
		},
		"MemorySummary": map[string]any{
			"TotalSystemMemoryGiB": float64(s.Memory.TotalBytes) / (1024 * 1024 * 1024),
			"Status":               statusBlock(domain.RedfishHealth(s.Memory.Health)),
		},
		"LogServices": map[string]any{
			"@odata.id": r.h.path("/redfish/v1/Systems/" + id + "/LogServices"),
		},
		"Oem": map[string]any{"RackPilot": map[string]any{"SyntheticTelemetry": true}},
	})
}

func (r *Router) chassisCollection(w http.ResponseWriter, _ *http.Request) {
	s := r.lookup()
	writeJSON(w, map[string]any{
		"@odata.context": r.h.path("/redfish/v1/$metadata#ChassisCollection.ChassisCollection"),
		"@odata.id":      r.h.path("/redfish/v1/Chassis"),
		"@odata.type":    schemaHost + "/ChassisCollection.ChassisCollection",
		"Members": []map[string]any{
			{"@odata.id": r.h.path("/redfish/v1/Chassis/" + s.ChassisID)},
		},
		"Members@odata.count": 1,
	})
}

func (r *Router) chassis(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.ChassisID {
		notFound(w)
		return
	}
	writeJSON(w, map[string]any{
		"@odata.id":   r.h.path("/redfish/v1/Chassis/" + id),
		"@odata.type": schemaHost + "/Chassis.v1_22_0.Chassis",
		"Id":          id,
		"Name":        "Synthetic Chassis",
		"ChassisType": "RackMount",
		"Status":      statusBlock(domain.RedfishHealth(s.Thermal.Health)),
		"Thermal":     map[string]any{"@odata.id": r.h.path("/redfish/v1/Chassis/" + id + "/Thermal")},
		"Power":       map[string]any{"@odata.id": r.h.path("/redfish/v1/Chassis/" + id + "/Power")},
	})
}

func (r *Router) thermal(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.ChassisID {
		notFound(w)
		return
	}
	temps := make([]map[string]any, 0, len(s.Thermal.Sensors))
	for _, t := range s.Thermal.Sensors {
		temps = append(temps, map[string]any{
			"MemberId":       t.ID,
			"Name":           t.Name,
			"ReadingCelsius": t.Celsius,
			"SensorNumber":   t.ID,
			"Status":         statusBlock("OK"),
			"ReadingUnits":   "Cel",
		})
	}
	fans := make([]map[string]any, 0, len(s.Thermal.Fans))
	for _, f := range s.Thermal.Fans {
		fans = append(fans, map[string]any{
			"MemberId":     f.ID,
			"Name":         f.Name,
			"Reading":      f.RPM,
			"ReadingUnits": "RPM",
			"Status":       statusBlock(domain.RedfishHealth(f.Health)),
		})
	}
	writeJSON(w, map[string]any{
		"@odata.id":    r.h.path("/redfish/v1/Chassis/" + id + "/Thermal"),
		"@odata.type":  schemaHost + "/Thermal.v1_7_0.Thermal",
		"Id":           "Thermal",
		"Temperatures": temps,
		"Fans":         fans,
	})
}

func (r *Router) power(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.ChassisID {
		notFound(w)
		return
	}
	psus := make([]map[string]any, 0, len(s.Power.PSUs))
	for _, p := range s.Power.PSUs {
		psus = append(psus, map[string]any{
			"MemberId":         p.ID,
			"Name":             "PSU " + p.ID,
			"PowerInputWatts":  p.InputWatts,
			"PowerOutputWatts": p.OutputWatts,
			"LineInputVoltage": 240,
			"Status":           statusBlock(domain.RedfishHealth(p.Health)),
		})
	}
	writeJSON(w, map[string]any{
		"@odata.id":   r.h.path("/redfish/v1/Chassis/" + id + "/Power"),
		"@odata.type": schemaHost + "/Power.v1_7_0.Power",
		"Id":          "Power",
		"PowerControl": []map[string]any{
			{
				"MemberId":           "0",
				"Name":               "System Power Control",
				"PowerConsumedWatts": s.Power.TotalWatts,
				"PowerCapacityWatts": 1600,
			},
		},
		"PowerSupplies": psus,
	})
}

func (r *Router) managerCollection(w http.ResponseWriter, _ *http.Request) {
	s := r.lookup()
	writeJSON(w, map[string]any{
		"@odata.id": r.h.path("/redfish/v1/Managers"),
		"Members": []map[string]any{
			{"@odata.id": r.h.path("/redfish/v1/Managers/" + s.ManagerID)},
		},
		"Members@odata.count": 1,
	})
}

func (r *Router) manager(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.ManagerID {
		notFound(w)
		return
	}
	writeJSON(w, map[string]any{
		"@odata.id":       r.h.path("/redfish/v1/Managers/" + id),
		"@odata.type":     schemaHost + "/Manager.v1_11_0.Manager",
		"Id":              id,
		"Name":            "Synthetic BMC",
		"ManagerType":     "BMC",
		"FirmwareVersion": "01.02.03-sim",
		"Status":          statusBlock(domain.RedfishHealth(s.SystemHealth)),
	})
}

func (r *Router) firmwareInventory(w http.ResponseWriter, _ *http.Request) {
	s := r.lookup()
	members := make([]map[string]any, 0, len(s.Firmware))
	for _, f := range s.Firmware {
		members = append(members, map[string]any{
			"@odata.id": r.h.path("/redfish/v1/UpdateService/FirmwareInventory/" + f.ID),
			"Id":        f.ID,
			"Name":      f.Name,
			"Version":   f.Version,
			"Status":    statusBlock("OK"),
			"Oem":       map[string]any{"RackPilot": map[string]any{"Synthetic": true}},
		})
	}
	writeJSON(w, map[string]any{
		"@odata.id":           r.h.path("/redfish/v1/UpdateService/FirmwareInventory"),
		"@odata.type":         schemaHost + "/SoftwareInventoryCollection.SoftwareInventoryCollection",
		"Members":             members,
		"Members@odata.count": len(members),
	})
}

func (r *Router) logServices(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.SystemID {
		notFound(w)
		return
	}
	writeJSON(w, map[string]any{
		"@odata.id": r.h.path("/redfish/v1/Systems/" + id + "/LogServices"),
		"Members": []map[string]any{
			{"@odata.id": r.h.path("/redfish/v1/Systems/" + id + "/LogServices/EventLog")},
		},
		"Members@odata.count": 1,
	})
}

func (r *Router) eventEntries(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	s := r.lookup()
	if id != s.SystemID {
		notFound(w)
		return
	}
	entries := make([]map[string]any, 0, len(s.Events))
	for _, e := range s.Events {
		entries = append(entries, map[string]any{
			"Id":        e.ID,
			"Created":   e.Created.Format("2006-01-02T15:04:05Z07:00"),
			"EntryType": "Event",
			"Severity":  e.Severity,
			"Message":   e.Message,
			"MessageId": e.MessageID,
			"Oem":       map[string]any{"RackPilot": map[string]any{"Synthetic": e.Synthetic}},
		})
	}
	writeJSON(w, map[string]any{
		"@odata.id":           r.h.path("/redfish/v1/Systems/" + id + "/LogServices/EventLog/Entries"),
		"@odata.type":         schemaHost + "/LogEntryCollection.LogEntryCollection",
		"Name":                "Event Log",
		"Members":             entries,
		"Members@odata.count": len(entries),
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}
