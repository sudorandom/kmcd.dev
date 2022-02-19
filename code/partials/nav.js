import PropTypes from 'prop-types';
import React, { Fragment } from 'react';

/**
 * The Nav component
 *
 * @disable-docs
 */
const Nav = ({ title, _nav, _pages, _ID }) => (
	<Fragment>
		<header>
			<div className="head-top">
				<a href="#" className="menu-btn"><span></span></a>
				<div className="top-menu">
					<ul>
					<li className={ _ID == "index" ? 'active' : null }><a href="/" className="lnk">index</a></li>
					{
						Object.keys(_nav["index"])
							.map(
								(page, i) =>(
									<li key={i} className={ _ID.startsWith(page) ? 'active' : null }><a href={_pages[page]._url} className="lnk">{_pages[page].title}</a></li>
								)
							)
					}
					</ul>
				</div>
			</div>
		</header>
	</Fragment>
);

Nav.defaultProps = {};

export default Nav;
