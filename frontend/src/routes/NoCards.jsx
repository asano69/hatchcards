import Button from "../components/Button";

// Session-summary screen shown when there are no cards due today.
export default function NoCards(props) {
  return (
    <div class="p-6 md:p-12">
      <h1 class="mb-9 text-center text-4xl md:mb-12 md:text-5xl">No cards due today.</h1>
      <div class="mt-12 text-center">
        <Button variant="" value="Home" onClick={props.onHome} />
      </div>
    </div>
  );
}
