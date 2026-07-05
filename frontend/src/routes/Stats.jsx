import { onMount, onCleanup, createSignal, Show } from "solid-js";
import embed from "vega-embed";
import NavBar from "../components/NavBar";
import pb from "../lib/pb";

const FORECAST_DAYS = 30;

async function fetchFutureDue() {
  return pb.collection("future_due_all").getFullList();
}

// Fill in the days that have no due cards with count=0, so the chart
// always shows a continuous 30-day range starting from today.
function buildDailySeries(records) {
  const counts = new Map();
  for (const r of records) {
    counts.set(Math.round(r.day_offset), r.count);
  }
  const series = [];
  for (let day = 0; day < FORECAST_DAYS; day++) {
    series.push({ day, count: counts.get(day) ?? 0 });
  }
  return series;
}

// Reads the same CSS custom properties the rest of the app uses, so the
// chart follows light/dark mode without needing its own color scheme.
function readThemeColors() {
  const style = getComputedStyle(document.documentElement);
  const get = (name, fallback) => style.getPropertyValue(name).trim() || fallback;
  return {
    text: get("--color-text", "#000000"),
    border: get("--color-border-soft", "#999999"),
    bar: get("--color-progress", "palegreen"),
  };
}

function buildSpec(series, colors) {
  return {
    $schema: "https://vega.github.io/schema/vega-lite/v6.json",
    width: "container",
    height: 300,
    // Transparent background so the chart blends into the page instead of
    // showing Vega's default white canvas in dark mode.
    background: "transparent",
    config: {
      axis: {
        labelColor: colors.text,
        titleColor: colors.text,
        domainColor: colors.border,
        tickColor: colors.border,
        gridColor: colors.border,
        gridOpacity: 0.2,
      },
      view: { stroke: "transparent" },
    },
    data: { values: series },
    mark: { type: "bar", color: colors.bar },
    encoding: {
      x: { field: "day", type: "ordinal", title: "Days from today" },
      y: { field: "count", type: "quantitative", title: "Cards due" },
      tooltip: [
        { field: "day", title: "Day" },
        { field: "count", title: "Cards due" },
      ],
    },
  };
}

export default function Stats() {
  let container;
  let view;
  let series = [];
  const [error, setError] = createSignal(null);

  // Re-renders the chart from whatever data is currently in `series`,
  // using the current theme colors. Does not fetch new data.
  const renderChart = async () => {
    view?.finalize();
    const result = await embed(container, buildSpec(series, readThemeColors()), { actions: false });
    view = result.view;
  };

  // Fetches the latest forecast data and re-renders the chart. Used both
  // on initial mount and when the user clicks Refresh in the nav bar.
  const load = async () => {
    try {
      series = buildDailySeries(await fetchFutureDue());
      setError(null);
    } catch (e) {
      setError(e);
      return;
    }
    await renderChart();
  };

  onMount(() => {
    load();

    // Re-render (not re-fetch) on OS theme change so the chart colors stay
    // in sync with the page's own dark/light mode.
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => renderChart();
    media.addEventListener("change", onChange);
    onCleanup(() => media.removeEventListener("change", onChange));
  });

  onCleanup(() => view?.finalize());

  return (
    <div class="mx-auto flex min-h-screen w-full max-w-3xl flex-col gap-6 bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      <NavBar onRefresh={load} />
      <h1 class="font-serif text-4xl">Review Forecast</h1>
      <Show when={error()}>
        <p class="text-[var(--color-border-soft)]">Failed to load forecast.</p>
      </Show>
      <div ref={container} class="w-full" />
    </div>
  );
}
