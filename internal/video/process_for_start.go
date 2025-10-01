package video

import "os/exec"

func ProcessVideoForFastStart(filePath string) (string, error) {
	procesingFilePath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", procesingFilePath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return procesingFilePath, nil
}
