# SpeedCam_Virtual

A proof of concept implemenation of the SpeedCam bandwidth usage algorithm for multipath communication in networks 
especially in SCION. Written in Go 1.9.

## Install

Install the dependencies via govendor (needs to be installed)

`govendor sync`

## Usage


### Run

The `local_network.go` contains the SpeedCam approach and can be run on a virtual SCION network like in a SCIONLab VM.

`go run local_network.go -scionDir=[PATH]`

or

build with `go build local_network.go` and run with `./local_network -scionDir=[PATH]`

### Necessary parameter

- `-scionDir=[PATH]`, where `[PATH]` points to a scion dir with its *gen* and *log* folder
path server requests.  Example: `-scionDir=/home/scion/go/src/scionproto/scion`

You can also run with the parameter `-h` or `--help` to print the help to console.

### Optional parameter

- `-cEpisodes=[INT]` - Set the amount of stored episodes(inspection cycles) before overriding old ones.

- `-cWDegree=[FLOAT]` - Set the weight of a nodes degree for its candidate score.

- `-cWCapacity=[FLOAT]` - Set the weight of a nodes capacity for its candidate score. Currently not supported (missing capacity info for link)

- `-cWSuccess=[FLOAT]` - Set the weight of a nodes success to identify congestion for its candidate score. Currently not supported.

- `-cWActivity=[FLOAT]` - Set the weight of a nodes activity for its candidate score. Currently simplified because of missing capacity information.

- `-verbose=[BOOLEAN]` - Enables/disables additional debug information. Default: enabled.

- `-resultDir=[String]` - If existing directory, the inspector will write the results to this directory as .JSON files. Default: '' (no output)

- `-scaleType=[String]` - Scaling of how many SpeedCams should be selected. Supported: **const**, **log** and **linear**. See `scaleParam` for more control.

- `-scaleParamFlag=[FLOAT]` - The parameter for the scale func. Base for **log**, factor for **linear** and the const for **const**. See `scaleType` for more information.

- `-cSpeedCamDiff=[INT]` - Additional(positive) or fewer(negative) SpeedCam to be selected. Will be added to result of `scalType`

- `-intervalStratFlag=[String]` - Strategy for waiting. Supported: **fixed**, **random** and **experience**. The last one uses the random configuration if there are too few time points in history.

- `-intervalMinFlag=[INT]` - Seconds to wait at minimum till next inspection.

- `-intervalMaxFlag=[INT]` - Seconds to wait at maximum till next inspection.
