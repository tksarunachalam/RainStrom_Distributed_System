package groupMembership

// timeout variable edit to 1 later.
const PingFrequency int = 6

// wait for t seconds before marking as failed
const failTimeout int = 5

// wait seconds for suspect mode
const SuspectWaitTimeout int = 4

// each node pings K neighbors
const KNeighbors int = 3

// introducer node Id. This process will join new members
const IntroducerNodeId = 2

const IntroducerPeerId = 789

// local::
// const IntroducerPeerId = 522
