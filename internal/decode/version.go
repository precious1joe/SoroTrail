package decode

// DecoderVersion is the hand-maintained version number of the decoder pipeline.
// It must be incremented deliberately when decoder output semantics change so that
// version-targeted replays can identify rows that need re-decoding.
const DecoderVersion = 1
