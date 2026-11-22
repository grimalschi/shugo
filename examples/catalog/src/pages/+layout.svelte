<script>
  import Footer from "../components/Footer.svelte"
  import Header from "../components/Header.svelte"
  import "../styles/main.css"

  let { children } = $props()
</script>

<Header />
<main>{@render children()}</main>
<Footer />
