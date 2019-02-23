//Package limits defines various hard limits in SQS.
package limits

// MaxBatchSize defines what could be the maximum size of the batch in SQS.
// It drives constraints in receive functions and in validators.
const MaxBatchSize = 10
