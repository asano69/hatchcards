import Button from "../components/Button";

// Session-summary screen shown after all cards in a session have been graded.
export default function NoCards(props) {
  return (
    <div class="p-6 md:p-12">
      <h1 class="mb-9 text-center text-4xl md:mb-12 md:text-5xl">No cards due today.</h1>
      <div class="shutdown-container">
        <button onClick={props.onHome}>Home</button>
      </div>
    </div>
  );
}
