package metrics

import (
	"fmt"
	"sync"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "rackpilot"

type Registrar struct {
	eng     *engine.Engine
	lastErr map[string]uint64
	errMu   sync.Mutex

	cpuTemp   *prometheus.GaugeVec
	cpuUtil   *prometheus.GaugeVec
	memTotal  *prometheus.GaugeVec
	memUsed   *prometheus.GaugeVec
	memErrors *prometheus.CounterVec
	fanRPM    *prometheus.GaugeVec
	fanHealth *prometheus.GaugeVec
	psuPower  *prometheus.GaugeVec
	psuHealth *prometheus.GaugeVec
	sysPower  *prometheus.GaugeVec
	sensorC   *prometheus.GaugeVec
	sysHealth *prometheus.GaugeVec
}

func NewRegistrar(reg prometheus.Registerer, eng *engine.Engine) *Registrar {
	r := &Registrar{eng: eng, lastErr: make(map[string]uint64)}
	labelsServer := []string{"server_id"}
	labelsCPU := []string{"server_id", "cpu_id"}
	labelsDIMM := []string{"server_id", "dimm_id"}
	labelsFan := []string{"server_id", "fan_id"}
	labelsPSU := []string{"server_id", "psu_id"}
	labelsSensor := []string{"server_id", "sensor_id"}

	help := "Synthetic BMC simulator metric (not physical hardware)."
	r.cpuTemp = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_cpu_temperature_celsius", Help: help}, labelsCPU)
	r.cpuUtil = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_cpu_utilization_percent", Help: help}, labelsCPU)
	r.memTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_memory_total_bytes", Help: help}, labelsServer)
	r.memUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_memory_used_bytes", Help: help}, labelsServer)
	r.memErrors = prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "bmc_memory_errors_total", Help: help + " Cumulative correctable+uncorrectable synthetic errors."}, labelsDIMM)
	r.fanRPM = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_fan_speed_rpm", Help: help}, labelsFan)
	r.fanHealth = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_fan_health_status", Help: help + " 0=OK,1=Warning,2=Critical."}, labelsFan)
	r.psuPower = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_psu_power_watts", Help: help + " Output watts."}, labelsPSU)
	r.psuHealth = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_psu_health_status", Help: help + " 0=OK,1=Warning,2=Critical."}, labelsPSU)
	r.sysPower = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_system_power_watts", Help: help}, labelsServer)
	r.sensorC = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_temperature_sensor_celsius", Help: help}, labelsSensor)
	r.sysHealth = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: "bmc_system_health_status", Help: help + " 0=OK,1=Warning,2=Critical."}, labelsServer)

	reg.MustRegister(
		r.cpuTemp, r.cpuUtil, r.memTotal, r.memUsed, r.memErrors,
		r.fanRPM, r.fanHealth, r.psuPower, r.psuHealth, r.sysPower, r.sensorC, r.sysHealth,
	)
	return r
}

func (r *Registrar) CollectFromEngine() {
	s := r.eng.Snapshot()
	sid := s.ServerID
	for _, c := range s.CPUs {
		l := prometheus.Labels{"server_id": sid, "cpu_id": c.ID}
		r.cpuTemp.With(l).Set(c.TempCelsius)
		r.cpuUtil.With(l).Set(c.UtilizationPct)
	}
	r.memTotal.WithLabelValues(sid).Set(float64(s.Memory.TotalBytes))
	r.memUsed.WithLabelValues(sid).Set(float64(s.Memory.UsedBytes))
	for _, d := range s.Memory.DIMMs {
		l := prometheus.Labels{"server_id": sid, "dimm_id": d.ID}
		total := d.CorrectableErrors + d.UncorrectableErrors
		key := fmt.Sprintf("%s:%s", sid, d.ID)
		r.errMu.Lock()
		prev := r.lastErr[key]
		if total > prev {
			r.memErrors.With(l).Add(float64(total - prev))
			r.lastErr[key] = total
		}
		r.errMu.Unlock()
	}
	for _, f := range s.Thermal.Fans {
		l := prometheus.Labels{"server_id": sid, "fan_id": f.ID}
		r.fanRPM.With(l).Set(f.RPM)
		r.fanHealth.With(l).Set(float64(domain.HealthToCode(f.Health)))
	}
	for _, p := range s.Power.PSUs {
		l := prometheus.Labels{"server_id": sid, "psu_id": p.ID}
		r.psuPower.With(l).Set(p.OutputWatts)
		r.psuHealth.With(l).Set(float64(domain.HealthToCode(p.Health)))
	}
	r.sysPower.WithLabelValues(sid).Set(s.SystemPowerWatts)
	for _, t := range s.Thermal.Sensors {
		l := prometheus.Labels{"server_id": sid, "sensor_id": t.ID}
		r.sensorC.With(l).Set(t.Celsius)
	}
	r.sysHealth.WithLabelValues(sid).Set(float64(domain.HealthToCode(s.SystemHealth)))
}
