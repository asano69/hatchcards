import Button from "../components/Button";

const statFont = { "font-family": '"SF Pro", sans-serif' };

function StatRow(props) {
  return (
    <tr class="border-b border-[var(--color-border-soft)]">
      <td class="py-3 whitespace-nowrap font-medium" style={statFont}>{props.label}</td>
      <td class="py-3 w-full pl-6 md:pl-12 tabular-nums" style={statFont}>{props.value}</td>
    </tr>
  );
}

// Session-summary screen shown after all cards in a session have been graded.
export default function Done(props) {
  const d = props.done;
  return (
    <div class="p-6 md:p-12">
      <h1 class="mb-9 text-center text-4xl md:mb-12 md:text-5xl">Session Completed 🎉</h1>
      <div class="mb-12 text-2xl">Reviewed {d.reviewed} cards in {d.durationSec} seconds.</div>
      <h2 class="mt-4 mb-4 text-3xl md:mt-6 md:mb-6 md:text-4xl border-b border-[var(--color-border-soft)]">Session Stats</h2>
      <div class="text-base md:text-[22px]">
        <table class="w-full border-collapse">
          <tbody>
            <StatRow label="Total Cards" value={d.total} />
            <StatRow label="Cards Reviewed" value={d.reviewed} />
            <StatRow label="Duration (seconds)" value={d.durationSec} />
          </tbody>
        </table>
      </div>
      <div class="shutdown-container">
        <Button variant="danger" value="Home" onClick={props.onHome} />
      </div>
    </div>
  );
}
