import '@wcj/dark-mode';
import styles from 'Header.module.less';
// @ts-ignore
import { ReactComponent } from './github.svg';
// @ts-ignore
import { ReactComponent as NginxLogo } from './nginx.svg';

export default function Header() {
  return (
    <div className={styles.header}>
      <NginxLogo />
      <div className={styles.title}>nginx config viewer</div>
      <div className={styles.filename}>readonly</div>
      <dark-mode permanent></dark-mode>
      <a href="https://github.com/jaywcjlove/nginx-editor" target="__blank" style={{ marginLeft: 6 }}>
        <ReactComponent />
      </a>
    </div>
  );
}
