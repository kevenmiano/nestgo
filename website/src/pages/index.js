import React from 'react';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import styles from './index.module.css';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

export default function Home() {
  const {siteConfig} = useDocusaurusContext();
  const releaseVersion = siteConfig.customFields?.releaseVersion;
  return <Layout title="Structured applications. Native Go." description="NestGo — experimental decorator-inspired Go compiler and HTTP runtime">
    <main>
      <header className="hero hero--primary">
        <div className="container">
          <img className={styles.mascot} src="/nestgo/img/nestgo-mascot.svg" alt="NestGo red mascot with speed trails" width="240" height="240"/>
          <div className={styles.heroEyebrow}>GO-NATIVE · COMPILER-FIRST · OPEN SOURCE</div>
          <h1 className="hero__title">Nest<span className={styles.accent}>Go</span></h1>
          {releaseVersion && <a className={styles.releaseBadge} href={`https://github.com/kevenmiano/nestgo/releases/tag/${encodeURIComponent(releaseVersion)}`}>
            <span className={styles.releaseDot} aria-hidden="true"/> Release {releaseVersion} · View changelog
          </a>}
          <p className="hero__subtitle">Structured applications. Native Go.</p>
          <p className={styles.lede}>Build modular HTTP applications with a decorator-inspired compiler, generated Go source, and the standard-library runtime.</p>
          <div className={styles.actions}>
            <Link className="button button--secondary button--lg" to="/docs/intro">Get started</Link>
            <Link className="button button--outline button--lg" style={{color:'white',borderColor:'white'}} to="/docs/roadmap">Explore roadmap</Link>
          </div>
          <p className={styles.notice}>Inspired by NestJS architecture. Built with native Go. Experimental, independent project.</p>
        </div>
      </header>
      <section className={styles.features}><div className="container"><div className="row">
        {[['Compiler-first','Compile .mgo annotations into ordinary Go source.'],['Go-native runtime','HTTP routing powered by the standard library.'],['Transparent maturity','Documented limitations and a public milestone roadmap.']].map(([title,description])=><div className="col col--4" key={title}><h2>{title}</h2><p>{description}</p></div>)}
      </div></div></section>
    </main>
  </Layout>;
}
