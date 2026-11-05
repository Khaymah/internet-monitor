import { CircleAlert, CloudDownload, ExternalLink, RefreshCw, Settings, Subtitles } from "lucide-react";
import { useRef, useState } from "react";
import { RouteLink } from "../components/RouteLink";
import { Banner, Button, Card } from "../components/ui";
import { tutorialSubtitlesUrl, tutorialVideoUrl } from "../content/tutorial";
import { LOCALES, useI18n } from "../lib/i18n";
import { useLive } from "../lib/live";

type Failure = "offline" | "unavailable" | null;

export function Tutorial() {
  const { t, lang } = useI18n();
  const version = useLive((s) => s.snapshot?.version);
  const [failure, setFailure] = useState<Failure>(null);
  const video = useRef<HTMLVideoElement>(null);
  const src = tutorialVideoUrl(version);

  const retry = () => {
    setFailure(null);
    video.current?.load();
    video.current?.play().catch(() => undefined);
  };

  return (
    <div className="doc-page">
      <header className="doc-hero">
        <p className="doc-eyebrow">{t("nav_tutorial")}</p>
        <h1>{t("tut_title")}</h1>
        <p className="secondary">{t("tut_intro")}</p>
      </header>

      <Card className="tutorial-card">
        <ul className="tutorial-notes">
          <li>
            <CloudDownload size={16} aria-hidden />
            <span>{t("tut_stream_note")}</span>
          </li>
          <li>
            <Subtitles size={16} aria-hidden />
            <span>{t("tut_subtitles_note")}</span>
          </li>
        </ul>
        {failure ? (
          <Banner
            tone="danger"
            icon={CircleAlert}
            actions={
              <>
                <Button size="sm" variant="ghost" icon={RefreshCw} onClick={retry}>
                  {t("retry")}
                </Button>
                <a className="btn btn-edge btn-sm" href={src} target="_blank" rel="noreferrer noopener">
                  {t("tut_open_github")}
                  <ExternalLink size={14} aria-hidden />
                  <span className="sr-only"> {t("opens_new_tab")}</span>
                </a>
              </>
            }
          >
            {t(failure === "offline" ? "tut_error_offline" : "tut_error")}
          </Banner>
        ) : null}
        {/* biome-ignore lint/a11y/useMediaCaption: a subtitle track for every app language is rendered below */}
        <video
          key={`${src}-${lang}`}
          ref={video}
          className="tutorial-video"
          src={src}
          controls
          preload="none"
          playsInline
          poster="/tutorial/poster.jpg"
          aria-label={t("tut_title")}
          onError={() => setFailure(navigator.onLine === false ? "offline" : "unavailable")}
          onPlaying={() => setFailure(null)}
        >
          {LOCALES.map((l) => (
            <track
              key={l.code}
              kind="subtitles"
              src={tutorialSubtitlesUrl(l.code)}
              srcLang={l.code}
              label={l.name}
              default={l.code === lang}
            />
          ))}
        </video>
      </Card>

      <div className="doc-cta">
        <RouteLink route="settings" className="btn btn-edge">
          <Settings size={16} aria-hidden />
          {t("tut_open_settings")}
        </RouteLink>
      </div>
    </div>
  );
}
