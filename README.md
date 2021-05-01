# Prometheus to Kafka/AQMP/NIFI
transfer open-metrics into air gapped environent with a tightly controlled diode. 

# reason
There are already many methods to transfer prometheus metrics to a centralized location. You can use prometheus federate method. but that will result in 1 large prometheus. You can use thanos or cortex

## write metrics
This software allows you to write metrics into several diode using AQMP, kafka or NIFI

That remote write API emits batched Snappy-compressed Protocol Buffer messages inside the body of an HTTP PUT request

## read metrics


### notes:
MINIO: Storage
GRAFANA: Visualizer
Zookeeper & KAFKA & Kafdrop: event streaming
Prometheus: Metric collector
thanos-sidecar: 
thanos-querier: query data 
thanos-store gateway:
thanos-receiver: 
thanos-compactor: downsampling and retention
thanos-ruler: 

todo: 
thanos-Query-Frontend
thanos-receiver

---------------------
                    -
    prometheus      -
    remote-write    -
                    -
-------->------------------------------------
  querier           -                       -
  prometheus+sidecar-                       - 
  prometheus+sidecar-                       -
  s3                -                       -
  grafana           - prometheus+sidecar    -
  rules             - prometheus+sidecar    -
  compactor         -------------------------
  receiver          -                       -
  receiver          - prometheus+sidecar    -
  storage-gateway   -                       -
--------->-----------------------------------
                    -
  prometheus        -
  remote-write      -
                    -
---------------------