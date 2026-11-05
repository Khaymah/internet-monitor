const RELEASES = "https://github.com/Khaymah/internet-monitor/releases";
const RELEASE_TAG = /^v\d+\.\d+\.\d+(-rc\.\d+)?$/;

export const TUTORIAL_FILE = "internet-monitor-tutorial.mp4";

export function tutorialVideoUrl(version: string | null | undefined): string {
  if (version && RELEASE_TAG.test(version)) return `${RELEASES}/download/${version}/${TUTORIAL_FILE}`;
  return `${RELEASES}/latest/download/${TUTORIAL_FILE}`;
}

export function tutorialSubtitlesUrl(lang: string): string {
  return `/tutorial/${lang}.vtt`;
}
