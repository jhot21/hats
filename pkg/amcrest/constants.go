package amcrest

const GeneralConfig string = "General"
const EncodeConfig string = "Encode"
const SnapConfig string = "Snap"

// TODO: ^ get complete list of config names

const MainFormat string = "MainFormat"
const SnapFormat string = "SnapFormat"
const ExtraFormat string = "ExtraFormat"

const EncodeTypeRegular int = 0
const EncodeTypeMotion int = 1
const EncodeTypeAlarm int = 2

const VideoBitRateParam string = "Video.BitRate"
const VideoBitRateControlParam string = "Video.BitRateControl"
const VideoCompressionParam string = "Video.Compression"
const VideoFPSParam string = "Video.FPS"
const VideoGOPParam string = "Video.GOP"
const VideoResolutionParam string = "Video.resolution"

// TODO: ^ get complete list of parameter names

const VideoEncodingMPEG4 string = "MPEG4"
const VideoEncodingMPEG2 string = "MPEG2"
const VideoEncodingMPEG1 string = "MPEG1"
const VideoEncodingMJPG string = "MJPG"
const VideoEncodingH263 string = "H263"
const VideoEncodingH264 string = "H264"
const VideoEncodingH265 string = "H265"

const VideoResolutionDefault string = "720x576"
const VideoResolution2560x1920 string = "2560x1920"
const VideoResolution5_1M = VideoResolution2560x1920
const VideoResolution1920x1080 string = "1920x1080"
const VideoResolution1080P = VideoResolution1920x1080

// TODO: ^ fill from table of named resolutions
