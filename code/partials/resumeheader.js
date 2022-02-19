import PropTypes from 'prop-types';
import React, { Fragment } from 'react';

import Nav from './Nav';

/**
 * The ResumeHeader component
 *
 * @disable-docs
 */
const ResumeHeader = ({ title, headerText, _ID, _pages, _nav }) => (
	<Fragment>
		<Nav _ID={_ID} _pages={_pages} _nav={_nav} />
		<div className="wrapper">
			<div className="section started">
				<div className="centrize full-width">
					<div className="vertical-center">
						<div className="started-content">
							<div className="h-title glitch-effect" data-text={ headerText }>{ headerText }</div>
							<div className="h-subtitle typing-subtitle">
								<p>Senior Software Engineer</p>
								<p>Based in Copenhagen</p>
								<p>This is my CV</p>
							</div>
							<span className="typed-subtitle"></span>
						</div>
					</div>
				</div>
				<a href="#" className="mouse_btn"><span className="ion ion-mouse"></span></a>
			</div>
		</div>
	</Fragment>
);

ResumeHeader.defaultProps = {
	"headerText": "I am Kevin McDonald"
};

export default ResumeHeader;
