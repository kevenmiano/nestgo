import React from 'react';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import styles from './index.module.css';

export default function Home() {
  return <Layout title="Structured applications. Native Go." description="NestGo — experimental decorator-inspired Go compiler and HTTP runtime">
    <main>
      <header className="hero hero--primary">
        <div className="container">
          <div className={styles.originalBrands}><img src="https://nestjs.com/img/logo-small.svg" alt="Official NestJS logo" width="76" height="76"/><span aria-hidden="true">+</span><img src="https://go.dev/images/go-logo-blue.svg" alt="Official Go language logo" width="130" height="76"/></div>
          <h1 className="hero__title">NestGo</h1>
          <p className="hero__subtitle">Structured applications. Native Go.</p>
          <p>Decorator-inspired development with generated Go code and a net/http runtime.</p>
          <div className={styles.actions}>
            <Link className="button button--secondary button--lg" to="/docs/intro">Get started</Link>
            <Link className="button button--outline button--lg" style={{color:'white',borderColor:'white'}} to="/docs/roadmap">Explore roadmap</Link>
          </div>
          <p className={styles.notice}>Inspired by NestJS architecture, built with Go. Independent project; not affiliated with NestJS or the Go team. Experimental proof of concept.</p>
        </div>
      </header>
      <section className={styles.features}><div className="container"><div className="row">
        {[['Compiler-first','Compile .mgo annotations into ordinary Go source.'],['Go-native runtime','HTTP routing powered by the standard library.'],['Transparent maturity','Documented limitations and a public milestone roadmap.']].map(([title,description])=><div className="col col--4" key={title}><h2>{title}</h2><p>{description}</p></div>)}
      </div></div></section>
    </main>
  </Layout>;
}
