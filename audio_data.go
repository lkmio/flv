package flv

import (
	"fmt"
	"github.com/lkmio/avformat/bufio"
	"github.com/lkmio/avformat/utils"
)

type SoundFormat int

const (
	SoundFormatPCMPlatform         = SoundFormat(0) // 按照创建文件的平台字序, 16-bit
	SoundFormatADPCM               = SoundFormat(1)
	SoundFormatMP3                 = SoundFormat(2)
	SoundFormatPCMLittle           = SoundFormat(3) // 如果SoundRate是8-bit无符号字节, 16-bit有符号字节
	SoundFormatNELLYMOSER16KHZMono = SoundFormat(4)
	SoundFormatNELLYMOSER8KHZMono  = SoundFormat(5)
	SoundFormatNELLYMOSER          = SoundFormat(6)
	SoundFormatG711A               = SoundFormat(7)
	SoundFormatG711B               = SoundFormat(8)
	SoundFormatAAC                 = SoundFormat(10)
	SoundFormatSpeex               = SoundFormat(11)
	SoundFormatMP38K               = SoundFormat(14)
	SoundFormatExHeader            = SoundFormat(9)
)

var (
	SupportedSampleRates = [4]int{5500, 11000, 22000, 44000}
)

type AudioData struct {
	SoundFormat SoundFormat
	Rate        int // 0-5.5k/1-11k/2-22k/3-44k
	Size        int // 0-8bit/1-16bit
	Type        int // 0-Mono/1-Stereo
}

func (a *AudioData) Marshal(dst []byte, sequenceHeader bool) int {
	_ = dst[0]

	dst[0] = byte(a.SoundFormat) << 4
	dst[0] |= byte(a.Rate & 0x3 << 2)
	dst[0] |= byte(a.Size & 0x1 << 1)
	dst[0] |= byte(a.Type & 0x1)

	if SoundFormatAAC == a.SoundFormat {
		if sequenceHeader {
			dst[1] = 0
		} else {
			dst[1] = 1
		}

		return 2
	}

	return 1
}

// Unmarshal 解析音频tag, 返回音频帧, 是否是头数据
func (a *AudioData) Unmarshal(data []byte) ([]byte, bool, error) {
	reader := bufio.NewBytesReader(data)
	flags, err := reader.ReadUint8()
	if err != nil {
		return nil, false, err
	}

	a.SoundFormat = SoundFormat(flags >> 4)
	a.Rate = int(flags >> 2 & 0x3)
	a.Size = int(flags >> 1 & 0x1)
	a.Type = int(flags & 0x1)

	if SoundFormatAAC == a.SoundFormat {
		pktType, err := reader.ReadUint8()
		if err != nil {
			return nil, false, err
		}

		return reader.RemainingBytes(), pktType == 0, nil
	} else {
		return reader.RemainingBytes(), false, err
	}
}

func AVCodecID2SoundFormat(id utils.AVCodecID, sampleRate int) (SoundFormat, error) {
	soundFormat, ok := SupportedCodecs[id]
	if !ok {
		return SoundFormat(-1), fmt.Errorf("unsupported audio codec: %v", id)
	} else if utils.AVCodecIdNELLYMOSER == soundFormat {
		if 8000 == sampleRate {
			soundFormat = SoundFormatNELLYMOSER8KHZMono
		} else if 16000 == sampleRate {
			soundFormat = SoundFormatNELLYMOSER16KHZMono
		} else {
			soundFormat = SoundFormatNELLYMOSER
		}
	}

	return soundFormat.(SoundFormat), nil
}

func SoundFormat2AVCodecID(format SoundFormat, sampleSize int) (utils.AVCodecID, error) {
	for avCodec, flvCodec := range SupportedCodecs {
		if flvCodec != format {
			continue
		}

		if SoundFormatPCMPlatform == format {
			if sampleSize == 8000 {
				return utils.AVCodecIdPCMU8, nil
			} else {
				//id = utils.AVCodecIdPCMS16BE
				return utils.AVCodecIdPCMS16LE, nil
			}
		} else if SoundFormatPCMLittle == format {
			if sampleSize == 8000 {
				return utils.AVCodecIdPCMU8, nil
			} else {
				return utils.AVCodecIdPCMS16LE, nil
			}
		}

		return avCodec, nil
	}

	if SoundFormatNELLYMOSER16KHZMono == format || SoundFormatNELLYMOSER8KHZMono == format || SoundFormatNELLYMOSER == format {
		return utils.AVCodecIdNELLYMOSER, nil
	}

	return utils.AVCodecIdNONE, fmt.Errorf("unknow sound format: %d", format)
}

func GetSampleRate(rate int) int {
	if rate < len(SupportedSampleRates) {
		return SupportedSampleRates[rate]
	}

	return -1
}

func NewAudioData(id utils.AVCodecID, sampleRate, sampleSize, channels int) (*AudioData, error) {
	format, err := AVCodecID2SoundFormat(id, sampleRate)
	if err != nil {
		return nil, err
	}

	// 0-5.5k/1-11k/2-22k/3-44k
	var rate int
	for i, v := range SupportedSampleRates {
		if v == sampleRate {
			rate = i
		}
	}

	// 0-8bit/1-16bit
	var size int
	if sampleSize == 8000 {
		size = 0
	} else {
		size = 1
	}

	// 0-Mono/1-Stereo
	var type_ int
	if channels > 1 {
		type_ = 1
	}

	data := &AudioData{
		SoundFormat: format,
		Rate:        rate,
		Size:        size,
		Type:        type_,
	}

	if SoundFormatAAC == data.SoundFormat {
		data.Rate = 3
		data.Type = 1
	} else if SoundFormatSpeex == data.SoundFormat {
		data.Rate = 0
		data.Size = 1
		data.Type = 0
	}

	return data, nil
}
