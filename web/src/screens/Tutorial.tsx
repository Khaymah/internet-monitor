import { CircleAlert, CirclePlay, CloudDownload, ExternalLink, RefreshCw, Subtitles } from "lucide-react";
import { useRef, useState } from "react";
import { RouteLink } from "../components/RouteLink";
import { Banner, Button, Card } from "../components/ui";
import {
  TOURS,
  type Tour,
  tourClock,
  tourOf,
  tourPosterUrl,
  tourSubtitlesUrl,
  tourVideoUrl,
} from "../content/tours";
import { formatDuration } from "../lib/format";
import { LOCALES, useI18n } from "../lib/i18n";
import { useLive } from "../lib/live";
import { useLocation } from "../lib/router";

type Failure = "offline" | "unavailable" | null;

function Length({ tour }: { tour: Tour }) {
  const { t, locale } = useI18n();
  if (tour.seconds <= 0) return null;
  return (
    <span className="tour-time num" title={t("tut_length", { duration: formatDuration(t, tour.seconds) })}>
      {tourClock(locale, tour.seconds)}
    </span>
  );
}

function Player({ tour }: { tour: Tour }) {
  const { t, lang } = useI18n();
  const version = useLive((s) => s.snapshot?.version);
  const [failure, setFailure] = useState<Failure>(null);
  const video = useRef<HTMLVideoElement>(null);
  const src = tourVideoUrl(tour.id, version);

  const retry = () => {
    setFailure(null);
    video.current?.load();
    video.current?.play().catch(() => undefined);
  };

  return (
    <Card className="tutorial-card" labelledBy="tour-now">
      <div className="tour-now-head">
        <h2 id="tour-now">{t(tour.title)}</h2>
        <Length tour={tour} />
      </div>
      <p className="secondary">{t(tour.desc)}</p>
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
        poster={tourPosterUrl(tour.id)}
        aria-label={t(tour.title)}
        onError={() => setFailure(navigator.onLine === false ? "offline" : "unavailable")}
        onPlaying={() => setFailure(null)}
      >
        {LOCALES.map((l) => (
          <track
            key={l.code}
            kind="subtitles"
            src={tourSubtitlesUrl(tour.id, l.code)}
            srcLang={l.code}
            label={l.name}
            default={l.code === lang}
          />
        ))}
      </video>
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
      <div className="row">
        <RouteLink route={tour.route} className="btn btn-edge btn-sm">
          {t("tut_open_page", { page: t(tour.page) })}
        </RouteLink>
      </div>
    </Card>
  );
}

export function Tutorial() {
  const { t } = useI18n();
  const { params } = useLocation();
  const current = tourOf(params.get("v"));

  return (
    <div className="doc-page">
      <header className="doc-hero">
        <p className="doc-eyebrow">{t("app_name")}</p>
        <h1>{t("tut_title")}</h1>
        <p className="secondary">{t("tut_intro")}</p>
      </header>

      <Player key={current.id} tour={current} />

      <section className="stack" aria-labelledby="tour-list">
        <h2 id="tour-list" className="tour-list-title">
          {t("tut_pick")}
        </h2>
        <ul className="tour-grid">
          {TOURS.map((tour) => {
            const selected = tour.id === current.id;
            return (
              <li key={tour.id}>
                <RouteLink route="tutorial" search={`?v=${tour.id}`} className="tour-card" current={selected}>
                  <span className="tour-thumb">
                    <img src={tourPosterUrl(tour.id)} alt="" loading="lazy" width={480} height={270} />
                    <CirclePlay size={34} aria-hidden className="tour-play" />
                    <Length tour={tour} />
                  </span>
                  <span className="tour-card-body">
                    <strong>{t(tour.title)}</strong>
                    {selected ? <span className="tour-badge">{t("tut_now")}</span> : null}
                    <span className="secondary">{t(tour.desc)}</span>
                  </span>
                </RouteLink>
              </li>
            );
          })}
        </ul>
      </section>
    </div>
  );
}
