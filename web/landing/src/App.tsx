import { Navbar } from "./components/Navbar";
import { Hero } from "./components/Hero";
import { LogoCloud } from "./components/LogoCloud";
import { HowItWorks } from "./components/HowItWorks";
import { Features } from "./components/Features";
import { Integrations } from "./components/Integrations";
import { CallToAction } from "./components/CallToAction";
import { Footer } from "./components/Footer";

function App() {
  return (
    <>
      <Navbar />
      <Hero />
      <LogoCloud />
      <HowItWorks />
      <Features />
      <Integrations />
      <CallToAction />
      <Footer />
    </>
  );
}

export default App;
